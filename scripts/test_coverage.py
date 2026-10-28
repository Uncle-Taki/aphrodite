#!/usr/bin/env python3
"""Run all Go tests with isolated Redis and enforce exact statement coverage."""

import argparse
from contextlib import contextmanager, ExitStack
import json
import os
from pathlib import Path
import shutil
import socket
import subprocess
import tempfile
import time
import uuid


@contextmanager
def integration_stack(root):
    """Use private databases and generated tokens, never the application's .env."""
    with tempfile.TemporaryDirectory(prefix="aphrodite-test-strapi-") as directory:
        env = os.environ.copy()
        env["APHRODITE_TEST_FIXTURE_DIR"] = directory
        # An explicit empty env file prevents Compose from reading the user's .env.
        empty_env = Path(directory) / "empty.env"
        empty_env.touch()
        compose = ["docker", "compose", "--env-file", str(empty_env),
                   "-p", "aphrodite-test-" + uuid.uuid4().hex[:12],
                   "-f", str(root / "build/docker-compose.test.yaml")]
        try:
            subprocess.run(compose + ["up", "-d", "--build", "--wait", "--wait-timeout", "180"],
                           cwd=root, env=env, check=True)
            def port(service, container_port):
                binding = subprocess.check_output(
                    compose + ["port", service, str(container_port)], cwd=root, env=env, text=True
                ).strip()
                return binding.rsplit(":", 1)[1]
            # The bootstrap writes host-readable credentials inside this private
            # (0700) temporary directory, which is removed after the tests.
            token_file = Path(directory) / "tokens.json"
            if not token_file.is_file():
                raise RuntimeError("Strapi did not create its test API tokens")
            yield {
                "aphrodite_TEST_POSTGRES_PORT": port("postgres", 5432),
                "aphrodite_TEST_STRAPI_URL": "http://127.0.0.1:" + port("strapi", 1337),
                "aphrodite_TEST_STRAPI_TOKEN_FILE": str(token_file),
            }
        except (subprocess.CalledProcessError, RuntimeError):
            subprocess.run(compose + ["logs", "--no-color", "--tail", "60"],
                           cwd=root, env=env, check=False)
            raise
        finally:
            subprocess.run(compose + ["down", "--volumes", "--remove-orphans"],
                           cwd=root, env=env, check=True)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output", type=Path, default=Path("/tmp/aphrodite-coverage"))
    parser.add_argument("--race", action="store_true")
    parser.add_argument("--integration", action="store_true",
                        help="also build and run isolated PostgreSQL and Strapi with Docker")
    args = parser.parse_args()
    root = Path(__file__).resolve().parents[1]
    output = args.output.resolve()
    output.mkdir(parents=True, exist_ok=True)
    redis = shutil.which("redis-server")
    if redis is None:
        parser.error("redis-server is required for the isolated Redis integration tests")

    with ExitStack() as resources, tempfile.TemporaryDirectory(prefix="aphrodite-test-redis-") as directory:
        integration_env = resources.enter_context(integration_stack(root)) if args.integration else {}
        with socket.socket() as reservation:
            reservation.bind(("127.0.0.1", 0))
            port = reservation.getsockname()[1]
        with open(Path(directory) / "redis.log", "w+") as log:
            server = subprocess.Popen(
                [redis, "--bind", "127.0.0.1", "--port", str(port),
                 "--save", "", "--appendonly", "no", "--dir", directory],
                stdout=log, stderr=subprocess.STDOUT,
            )
            try:
                deadline = time.monotonic() + 10
                while True:
                    if server.poll() is not None:
                        log.seek(0)
                        raise RuntimeError(f"Redis startup failed: {log.read()}")
                    try:
                        with socket.create_connection(("127.0.0.1", port), timeout=0.1):
                            break
                    except OSError:
                        if time.monotonic() >= deadline:
                            raise RuntimeError("Redis startup timed out")
                        time.sleep(0.05)
                env = os.environ.copy()
                # Live services are explicitly opted into through --integration.
                for key in ("aphrodite_TEST_POSTGRES_PORT", "aphrodite_TEST_STRAPI_URL",
                            "aphrodite_TEST_STRAPI_TOKEN_FILE"):
                    env.pop(key, None)
                env.update(integration_env)
                env["aphrodite_TEST_REDIS_ADDR"] = f"127.0.0.1:{port}"
                env["aphrodite_TEST_REDIS_PASSWORD"] = ""
                command = ["go", "test", "-count=1", "-covermode=atomic", "-coverpkg=./...",
                           f"-coverprofile={output / 'coverage.out'}"]
                if args.race:
                    command.append("-race")
                test_log = output / "tests.jsonl"
                with test_log.open("w") as log:
                    result = subprocess.run(command + ["-json", "./..."], cwd=root,
                                            env=env, stdout=log, stderr=subprocess.STDOUT)
                events = []
                for line in test_log.read_text().splitlines():
                    try:
                        events.append(json.loads(line))
                    except json.JSONDecodeError:
                        print(line, flush=True)
                failed_packages = {event["Package"] for event in events
                                   if event.get("Action") == "fail"}
                for event in events:
                    if event.get("Action") == "output" and (
                        event["Package"] in failed_packages or (
                            not event.get("Test") and event["Output"].startswith(("ok  ", "?", "\t"))
                        )
                    ):
                        print(event["Output"], end="", flush=True)
                result.check_returncode()
                passed = {(event["Package"], event["Test"]) for event in events
                          if event.get("Action") == "pass" and event.get("Test")}
                skipped = [event["Test"] for event in events
                           if event.get("Action") == "skip" and event.get("Test")]
                print(f"Tests: {sum('/' not in name for _, name in passed)} top-level passed; "
                      f"{len(skipped)} skipped", flush=True)
                required = {"TestClientLifecycle_GatedRedis", "TestQueue_EnqueuesTaskInRedis"}
                if args.integration:
                    required.update({"TestAPI_LivePostgresRedisAndStrapi",
                                     "TestClient_LiveStrapiCredentialsAndDocumentLifecycle"})
                passed_names = {name for _, name in passed}
                if not required.issubset(passed_names):
                    raise RuntimeError(f"Required integration tests did not pass: {required - passed_names}")
            finally:
                server.terminate()
                try:
                    server.wait(timeout=5)
                except subprocess.TimeoutExpired:
                    server.kill()
                    server.wait()

    # Cross-package instrumentation may repeat blocks. A block is covered if
    # any test package executed it; count its statements only once.
    blocks = {}
    for line in (output / "coverage.out").read_text().splitlines()[1:]:
        location, statements, count = line.split()
        key = (location, int(statements))
        blocks[key] = blocks.get(key, False) or int(count) > 0
    total = sum(statements for _, statements in blocks)
    if not total:
        raise RuntimeError("The coverage profile contains no statements")
    covered = sum(statements for (_, statements), hit in blocks.items() if hit)
    subprocess.run(["go", "tool", "cover", f"-html={output / 'coverage.out'}",
                    f"-o={output / 'coverage.html'}"], cwd=root, check=True)
    print(f"Statement coverage: {covered}/{total} ({100 * covered / total:.2f}%)", flush=True)
    print(f"HTML report: {output / 'coverage.html'}", flush=True)
    missing = [location for (location, statements), hit in blocks.items() if statements and not hit]
    if missing:
        print("Uncovered blocks:\n" + "\n".join(missing), flush=True)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

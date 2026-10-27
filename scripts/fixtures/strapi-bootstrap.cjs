// Mounted only by docker-compose.test.yaml into its disposable Strapi instance.
// Generated credentials stay in the runner's temporary directory.
const fs = require('node:fs');

module.exports = {
  register() {},
  async bootstrap({ strapi }) {
    const service = strapi.admin.services['api-token'];
    const full = await service.create({
      name: 'coverage-full', type: 'full-access', lifespan: null,
    });
    const readOnly = await service.create({
      name: 'coverage-read-only', type: 'read-only', lifespan: null,
    });
    fs.writeFileSync('/test-fixtures/tokens.json', JSON.stringify({
      full: full.accessKey, readOnly: readOnly.accessKey,
    }), { mode: 0o644 });
  },
};

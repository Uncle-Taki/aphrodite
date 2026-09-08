export default ({ env }: { env: any }) => ({
  rest: {
    defaultLimit: env.int('STRAPI_API_DEFAULT_LIMIT', 25),
    maxLimit: env.int('STRAPI_API_MAX_LIMIT', 100),
    withCount: true,
  },
});

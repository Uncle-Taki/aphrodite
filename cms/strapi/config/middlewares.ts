export default ({ env }: { env: any }) => {
  const publicBaseUrl = env('S3_PUBLIC_BASE_URL', '');
  const mediaOrigins = publicBaseUrl ? [new URL(publicBaseUrl).origin] : [];

  return [
    'strapi::logger',
    'strapi::errors',
    {
      name: 'strapi::security',
      config: {
        contentSecurityPolicy: {
          useDefaults: true,
          directives: {
            'connect-src': ["'self'", 'https:', ...mediaOrigins],
            'img-src': ["'self'", 'data:', 'blob:', 'market-assets.strapi.io', ...mediaOrigins],
            'media-src': ["'self'", 'data:', 'blob:', ...mediaOrigins],
            // Local and IP-based deployments may serve media over HTTP.
            upgradeInsecureRequests: null,
          },
        },
      },
    },
    'strapi::cors',
    'strapi::query',
    'strapi::body',
    'strapi::session',
    'strapi::favicon',
    'strapi::public',
  ];
};

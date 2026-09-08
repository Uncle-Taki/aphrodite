export default ({ env }: { env: any }) => ({
  i18n: {
    enabled: true,
    config: {
      defaultLocale: env('STRAPI_PLUGIN_I18N_INIT_LOCALE_CODE', 'fa'),
      locales: ['fa', 'en'],
    },
  },
  upload: {
    config: {
      provider: env('STRAPI_UPLOAD_PROVIDER', 'aws-s3'),
      providerOptions: {
        baseUrl: env('S3_PUBLIC_BASE_URL', ''),
        rootPath: env('S3_ROOT_PATH', ''),
        s3Options: {
          credentials: {
            accessKeyId: env('S3_ACCESS_KEY_ID'),
            secretAccessKey: env('S3_SECRET_ACCESS_KEY'),
          },
          region: env('S3_REGION', 'us-east-1'),
          endpoint: env('S3_ENDPOINT', undefined),
          forcePathStyle: env.bool('S3_FORCE_PATH_STYLE', false),
          params: {
            Bucket: env('S3_BUCKET'),
          },
        },
      },
    },
  },
});

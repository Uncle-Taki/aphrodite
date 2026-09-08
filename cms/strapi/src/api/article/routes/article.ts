export default {
  type: 'content-api',
  routes: [
    {
      method: 'GET',
      path: '/articles',
      handler: 'api::article.article.find',
    },
    {
      method: 'GET',
      path: '/articles/:id',
      handler: 'api::article.article.findOne',
    },
    {
      method: 'POST',
      path: '/articles',
      handler: 'api::article.article.create',
    },
    {
      method: 'PUT',
      path: '/articles/:id',
      handler: 'api::article.article.update',
    },
    {
      method: 'DELETE',
      path: '/articles/:id',
      handler: 'api::article.article.delete',
    },
  ],
};

export default {
  type: 'content-api',
  routes: [
    { method: 'GET', path: '/products', handler: 'api::product.product.find' },
    { method: 'GET', path: '/products/:id', handler: 'api::product.product.findOne' },
    { method: 'POST', path: '/products', handler: 'api::product.product.create' },
    { method: 'PUT', path: '/products/:id', handler: 'api::product.product.update' },
    { method: 'DELETE', path: '/products/:id', handler: 'api::product.product.delete' },
  ],
};

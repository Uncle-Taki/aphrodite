export default {
  type: 'content-api',
  routes: [
    { method: 'GET', path: '/main-categories', handler: 'api::main-category.main-category.find' },
    { method: 'GET', path: '/main-categories/:id', handler: 'api::main-category.main-category.findOne' },
    { method: 'POST', path: '/main-categories', handler: 'api::main-category.main-category.create' },
    { method: 'PUT', path: '/main-categories/:id', handler: 'api::main-category.main-category.update' },
    { method: 'DELETE', path: '/main-categories/:id', handler: 'api::main-category.main-category.delete' },
  ],
};

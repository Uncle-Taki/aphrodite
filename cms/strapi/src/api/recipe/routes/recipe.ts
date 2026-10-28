export default {
  type: 'content-api',
  routes: [
    { method: 'GET', path: '/recipes', handler: 'api::recipe.recipe.find' },
    { method: 'GET', path: '/recipes/:id', handler: 'api::recipe.recipe.findOne' },
    { method: 'POST', path: '/recipes', handler: 'api::recipe.recipe.create' },
    { method: 'PUT', path: '/recipes/:id', handler: 'api::recipe.recipe.update' },
    { method: 'DELETE', path: '/recipes/:id', handler: 'api::recipe.recipe.delete' },
  ],
};

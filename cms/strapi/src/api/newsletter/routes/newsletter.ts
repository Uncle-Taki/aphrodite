export default {
  type: 'content-api',
  routes: [
    { method: 'GET', path: '/newsletters', handler: 'api::newsletter.newsletter.find' },
    { method: 'GET', path: '/newsletters/:id', handler: 'api::newsletter.newsletter.findOne' },
    { method: 'POST', path: '/newsletters', handler: 'api::newsletter.newsletter.create' },
    { method: 'PUT', path: '/newsletters/:id', handler: 'api::newsletter.newsletter.update' },
    { method: 'DELETE', path: '/newsletters/:id', handler: 'api::newsletter.newsletter.delete' },
  ],
};

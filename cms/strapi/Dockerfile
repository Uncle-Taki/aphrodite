FROM node:22-bookworm-slim AS dependencies
WORKDIR /opt/app
COPY package*.json ./
RUN npm install

FROM node:22-bookworm-slim AS runtime
WORKDIR /opt/app
ENV NODE_ENV=production
COPY --from=dependencies /opt/app/node_modules ./node_modules
COPY . .
RUN npm run build
# TypeScript emits the JS files, while Strapi loads content-type and component
# schemas from the compiled src tree at runtime.
RUN mkdir -p dist/src/api dist/src/components \
    && cp -R src/api/. dist/src/api/ \
    && cp -R src/components/. dist/src/components/ \
    && find dist/src/api dist/src/components -type f -name '*.ts' -delete
EXPOSE 1337
CMD ["npm", "run", "start"]

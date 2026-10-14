# Frontend SvelteKit (adapter-node). Build dari folder App/frontend:
#   docker build -t spkzakat-web frontend
FROM node:24-alpine AS build
WORKDIR /src
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
RUN npm run build && npm prune --omit=dev

FROM node:24-alpine
WORKDIR /app
COPY --from=build /src/build ./build
COPY --from=build /src/node_modules ./node_modules
COPY --from=build /src/package.json ./
ENV NODE_ENV=production PORT=3000 HOST=0.0.0.0
# Image node sudah menyediakan user non-root `node`.
USER node
EXPOSE 3000
CMD ["node", "build"]

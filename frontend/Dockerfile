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
# BODY_SIZE_LIMIT: bawaan adapter-node 512 KB terlalu kecil untuk unggah gambar
# sampul program (maks 2 MB, dibatasi lagi di SvelteKit dan Go).
ENV NODE_ENV=production PORT=3000 HOST=0.0.0.0 BODY_SIZE_LIMIT=3M
# Image node sudah menyediakan user non-root `node`.
USER node
EXPOSE 3000
CMD ["node", "build"]

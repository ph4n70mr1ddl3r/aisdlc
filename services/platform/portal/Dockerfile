# syntax=docker/dockerfile:1
# M0 STUB — replaced by the real service in its milestone (see ROADMAP.md).
FROM node:20-alpine
WORKDIR /app
COPY server.js .
EXPOSE 3000
HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=5 CMD wget -qO- http://localhost:3000/healthz >/dev/null 2>&1 || exit 1
CMD ["node", "server.js"]

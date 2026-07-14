# syntax=docker/dockerfile:1
# M0 STUB — replaced by the real service in its milestone (see ROADMAP.md).
FROM python:3.12-slim
RUN groupadd -r app && useradd -r -g app app && \
    apt-get update && apt-get install -y --no-install-recommends wget && \
    rm -rf /var/lib/apt/lists/*
ENV PYTHONDONTWRITEBYTECODE=1 PYTHONUNBUFFERED=1
WORKDIR /app
COPY shared/stub-py/server.py /app/_stub.py
COPY services/core/workflow-engine/server.py .
USER app
EXPOSE 8000
HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=5 CMD wget -qO- http://localhost:8000/healthz >/dev/null 2>&1 || exit 1
CMD ["python", "server.py"]

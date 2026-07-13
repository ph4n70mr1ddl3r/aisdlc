# syntax=docker/dockerfile:1
# M0 STUB — replaced by the real service in its milestone (see ROADMAP.md).
FROM python:3.12-slim
WORKDIR /app
COPY shared/stub-py/server.py /app/_stub.py
COPY services/core/workflow-engine/server.py .
EXPOSE 8000
HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=5 CMD python -c "import urllib.request,sys; sys.exit(0 if urllib.request.urlopen('http://localhost:8000/healthz', timeout=2).status == 200 else 1)"
CMD ["python", "server.py"]

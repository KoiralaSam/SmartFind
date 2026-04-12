FROM python:3.11-slim

WORKDIR /app

COPY services/predictive-analytics-agent/requirements.txt ./

RUN python3 -m venv /app/venv
RUN /app/venv/bin/pip install --upgrade pip setuptools wheel
RUN /app/venv/bin/pip install --no-cache-dir -r requirements.txt

COPY services/predictive-analytics-agent ./

EXPOSE 8092

CMD ["/app/venv/bin/python", "-m", "uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8092"]

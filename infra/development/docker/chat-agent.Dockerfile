FROM python:3.11-slim

WORKDIR /app

COPY services/chat-agent/requirements.txt ./

RUN pip install --no-cache-dir -r requirements.txt

COPY services/chat-agent ./

EXPOSE 8090

CMD ["python", "-m", "uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8090"]

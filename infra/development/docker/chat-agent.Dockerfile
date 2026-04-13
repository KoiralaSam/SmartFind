FROM python:3.11-slim

WORKDIR /app

COPY services/chat-agent/requirements.txt ./ 

# Create virtual environment
RUN python3 -m venv /app/venv

# Upgrade pip in the virtual environment
RUN /app/venv/bin/pip install --upgrade pip setuptools wheel

# Install requirements using venv's pip
RUN /app/venv/bin/pip install --no-cache-dir -r requirements.txt

COPY services/chat-agent ./

# Copy generated gRPC python files (shared/proto_py) into the image
COPY shared/proto_py ./shared/proto_py

EXPOSE 8090

CMD ["/app/venv/bin/python", "-m", "uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8090"]

# Xagent - Ultra-lightweight personal AI agent
# OpenClaw-RL integration: Dockerfile for the RL training server
# License: MIT

FROM nvidia/cuda:12.9.0-devel-ubuntu24.04

# Install Python and dependencies
RUN apt-get update && apt-get install -y \
    python3.12 python3.12-venv python3-pip curl git \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy the OpenClaw-RL code
COPY requirements.txt .
RUN pip3 install --no-cache-dir -r requirements.txt

# Copy Slime framework and RL server code
COPY slime/ ./slime/
COPY openclaw-rl/ ./openclaw-rl/
COPY openclaw-opd/ ./openclaw-opd/

# Install Slime
RUN cd slime && pip3 install -e .

# Default entrypoint: start the Binary RL server
# Override with docker compose command for OPD
ENTRYPOINT ["bash"]
CMD ["./openclaw-rl/run_qwen3_4b_openclaw_rl.sh"]

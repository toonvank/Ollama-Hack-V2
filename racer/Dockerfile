FROM rust:1.85-bookworm AS builder
WORKDIR /app
COPY Cargo.toml Cargo.lock* ./
COPY crates ./crates
RUN cargo build --release --bin ollama-racer

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*
COPY --from=builder /app/target/release/ollama-racer /usr/local/bin/ollama-racer
ENV RACER_BIND=0.0.0.0:8787
EXPOSE 8787
ENTRYPOINT ["/usr/local/bin/ollama-racer"]
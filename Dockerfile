FROM golang:1.24-alpine AS go-builder
WORKDIR /build
COPY go.mod go.sum* ./
RUN go mod download 2>/dev/null || true
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /vc ./cmd/vc

FROM debian:trixie-slim AS ffmpeg-builder

ARG FFMPEG_VERSION=7.1.1
ARG VMAF_VERSION=3.0.0

RUN apt-get update && apt-get install -y --no-install-recommends \
    autoconf automake build-essential cmake curl dpkg-dev g++ git libtool \
    meson nasm ninja-build pkg-config yasm xxd \
    libx264-dev libx265-dev libnuma-dev libvpx-dev \
    libmp3lame-dev libopus-dev libvorbis-dev libass-dev libfreetype-dev \
    libgnutls28-dev libva-dev libvdpau-dev zlib1g-dev \
    ca-certificates && \
    rm -rf /var/lib/apt/lists/*

RUN curl -sL https://github.com/Netflix/vmaf/archive/refs/tags/v${VMAF_VERSION}.tar.gz | \
    tar xz -C /tmp && \
    cd /tmp/vmaf-${VMAF_VERSION}/libvmaf && \
    meson setup build --buildtype=release --default-library=static \
        -Denable_tests=false -Denable_docs=false \
        -Dbuilt_in_models=true -Denable_float=true \
        --prefix=/usr/local && \
    ninja -C build && \
    ninja -C build install && \
    rm -rf /tmp/vmaf-${VMAF_VERSION}

RUN ARCH=$(dpkg-architecture -qDEB_HOST_MULTIARCH) && \
    curl -sL https://ffmpeg.org/releases/ffmpeg-${FFMPEG_VERSION}.tar.bz2 | \
    tar xj -C /tmp && \
    cd /tmp/ffmpeg-${FFMPEG_VERSION} && \
    PKG_CONFIG_PATH=/usr/local/lib/${ARCH}/pkgconfig:/usr/local/lib/pkgconfig \
    ./configure \
        --prefix=/usr/local \
        --enable-gpl \
        --enable-version3 \
        --enable-nonfree \
        --enable-libvmaf \
        --enable-libx264 \
        --enable-libx265 \
        --enable-libvpx \
        --enable-libmp3lame \
        --enable-libopus \
        --enable-libvorbis \
        --enable-libass \
        --enable-libfreetype \
        --enable-gnutls \
        --extra-libs="-lpthread -lm -lstdc++" && \
    make -j$(nproc) && \
    make install && \
    rm -rf /tmp/ffmpeg-${FFMPEG_VERSION}

FROM debian:trixie-slim
RUN apt-get update && apt-get install -y --no-install-recommends \
    libx264-164 libx265-215 libvpx9 libmp3lame0 \
    libopus0 libvorbisenc2 libass9 libfreetype6 \
    libva2 libva-drm2 libva-x11-2 \
    libgnutls30t64 libnuma1 libvdpau1 libdrm2 && \
    rm -rf /var/lib/apt/lists/*
COPY --from=ffmpeg-builder /usr/local/bin/ffmpeg /usr/local/bin/ffmpeg
COPY --from=ffmpeg-builder /usr/local/bin/ffprobe /usr/local/bin/ffprobe
COPY --from=ffmpeg-builder /usr/local/lib/ /usr/local/lib/
RUN ldconfig
COPY --from=go-builder /vc /usr/local/bin/vc
ENTRYPOINT ["vc"]

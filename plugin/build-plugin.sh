#!/bin/bash
set -e

PLUGIN_NAME=${1:-sistemica/docker-volume-manager-csi}
PLUGIN_TAG=${2:-latest}

echo "Building Docker managed plugin: $PLUGIN_NAME:$PLUGIN_TAG"

# Step 1: Build the plugin image
echo "Step 1: Building plugin Docker image..."
docker build -t ${PLUGIN_NAME}-builder:${PLUGIN_TAG} -f plugin/Dockerfile .

# Step 2: Create a container and export rootfs
echo "Step 2: Creating container and exporting rootfs..."
CONTAINER_ID=$(docker create ${PLUGIN_NAME}-builder:${PLUGIN_TAG})
mkdir -p plugin/rootfs
docker export ${CONTAINER_ID} | tar -x -C plugin/rootfs
docker rm ${CONTAINER_ID}

# Step 3: Remove existing plugin if it exists
echo "Step 3: Removing existing plugin (if any)..."
docker plugin rm -f ${PLUGIN_NAME}:${PLUGIN_TAG} 2>/dev/null || true

# Step 4: Create the plugin
echo "Step 4: Creating Docker plugin..."
docker plugin create ${PLUGIN_NAME}:${PLUGIN_TAG} ./plugin

# Step 5: Clean up rootfs
echo "Step 5: Cleaning up..."
rm -rf plugin/rootfs

echo "Plugin created successfully: ${PLUGIN_NAME}:${PLUGIN_TAG}"
echo ""
echo "To enable the plugin, run:"
echo "  docker plugin enable ${PLUGIN_NAME}:${PLUGIN_TAG}"
echo ""
echo "To install with specific settings:"
echo "  docker plugin set ${PLUGIN_NAME}:${PLUGIN_TAG} MANAGER_URL=http://volume-manager:9789"
echo "  docker plugin enable ${PLUGIN_NAME}:${PLUGIN_TAG}"

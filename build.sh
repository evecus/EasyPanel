#!/bin/bash
# EasyPanel Build Script
# Builds the Vue 3 frontend and compiles the Go binary

set -e

echo "📦 Installing frontend dependencies..."
cd frontend
npm install

echo "🔨 Building Vue 3 frontend..."
npm run build

echo "🐹 Compiling Go backend..."
cd ..
go build -o easypanel .

echo ""
echo "✅ Build complete! Run: ./easypanel"
echo "   Default credentials: admin / admin"

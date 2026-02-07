# Edge Mesh Web Console

Web-based control console for the Edge Mesh distributed orchestration platform.

## Tech Stack

- **React 18** + **TypeScript** - UI framework
- **Vite** - Build tool and dev server
- **shadcn/ui** - Component library
- **Tailwind CSS** - Styling
- **Framer Motion** - Animations
- **React Query** - Server state management

## Getting Started

```sh
# Install dependencies
npm install

# Start development server (http://localhost:8080)
npm run dev

# Build for production
npm run build

# Run tests
npm test
```

## Console Pages

| Page | Route | Purpose |
|------|-------|---------|
| Connect | `/console/connect` | Gateway connection |
| Dashboard | `/console/dashboard` | Overview and KPIs |
| Devices | `/console/devices` | Device management |
| Run | `/console/run` | Command execution |
| Chat | `/console/chat` | AI assistant |
| Jobs | `/console/jobs` | Job management |
| Settings | `/console/settings` | Configuration |

## Backend Integration

The console connects to the Edge Mesh orchestrator server:
- **gRPC**: Port 50051
- **HTTP**: Port 8080

Configure the server address on the Connect page.

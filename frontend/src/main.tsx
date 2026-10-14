import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import App from "./App";
import { ToastHost } from "./components/ui";
import "./styles.css";

// Go sends "window:focus" when the window comes forward, so nothing is refetched on a timer.
const queryClient = new QueryClient({
  defaultOptions: { queries: { refetchOnWindowFocus: false, retry: 1, staleTime: 10_000 } },
});

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <ToastHost>
        <App />
      </ToastHost>
    </QueryClientProvider>
  </StrictMode>,
);

import { useEffect } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Events } from "@wailsio/runtime";
import * as Service from "../bindings/github.com/AlBannaTechno/claude-sessions-list/internal/app/service";
export type { Entry, Group, Listing, Summary, Transcript } from "../bindings/github.com/AlBannaTechno/claude-sessions-list/internal/sessions/models";

export { Service };

export const keys = {
  listing: ["listing"] as const,
  transcript: (id: string) => ["transcript", id] as const,
};

export function useListing() {
  return useQuery({ queryKey: keys.listing, queryFn: () => Service.ListGroups() });
}

export function useTranscript(id: string | null, enabled: boolean) {
  return useQuery({ queryKey: keys.transcript(id ?? ""), queryFn: () => Service.LoadTranscript(id!), enabled: !!id && enabled, staleTime: 5_000 });
}

/** Coming back to the window reads the sessions again (Go sends "window:focus"), so new ones show up. */
export function useRefreshOnFocus() {
  const qc = useQueryClient();
  useEffect(() => {
    const off = Events.On("window:focus", () => {
      void qc.invalidateQueries({ queryKey: keys.listing });
      void qc.invalidateQueries({ queryKey: ["transcript"] });
    });
    return () => off();
  }, [qc]);
}

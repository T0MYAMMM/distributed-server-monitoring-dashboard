"use client";

import { useEffect, useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import toast from "react-hot-toast";
import {
  DndContext,
  closestCenter,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors,
  type DragEndEvent,
} from "@dnd-kit/core";
import {
  SortableContext,
  arrayMove,
  sortableKeyboardCoordinates,
  useSortable,
  verticalListSortingStrategy,
} from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { GripVertical, ServerOff, Trash2 } from "lucide-react";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Button } from "@/components/ui/button";
import { StatusBadge } from "@/components/shared/StatusBadge";
import { CopyButton } from "@/components/shared/CopyButton";
import { EmptyState } from "@/components/shared/EmptyState";
import { API_URL } from "@/config/config";
import { deleteServer, setServerOrder } from "@/lib/api/client";
import { serversKey } from "@/lib/hooks/useServers";
import type { Server } from "@/lib/api/types";

function linuxCommand(name: string): string {
  return `curl -fsSL ${API_URL}/download/install_agent.sh | sudo bash -s ${name} ${API_URL}`;
}
function windowsCommand(name: string): string {
  return `iwr ${API_URL}/download/install_agent.ps1 -OutFile install_agent.ps1; .\\install_agent.ps1 -NodeName ${name} -ServerUrl ${API_URL}`;
}

function SortableRow({
  server,
  onDelete,
}: {
  server: Server;
  onDelete: (s: Server) => void;
}) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id: server.id,
  });
  const style: React.CSSProperties = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.6 : 1,
  };
  return (
    <TableRow ref={setNodeRef} style={style} className={isDragging ? "bg-muted" : undefined}>
      <TableCell className="w-8">
        <button
          {...attributes}
          {...listeners}
          aria-label={`Reorder ${server.name}`}
          className="cursor-grab touch-none rounded p-1 text-muted-foreground hover:bg-accent active:cursor-grabbing"
        >
          <GripVertical className="h-4 w-4" />
        </button>
      </TableCell>
      <TableCell className="font-medium">{server.name}</TableCell>
      <TableCell>
        <StatusBadge status={server.status} />
      </TableCell>
      <TableCell>
        <div className="flex gap-2">
          <CopyButton value={linuxCommand(server.name)} label="Linux" />
          <CopyButton value={windowsCommand(server.name)} label="Windows" />
        </div>
      </TableCell>
      <TableCell className="text-right">
        <Button
          variant="ghost"
          size="icon"
          aria-label={`Delete ${server.name}`}
          onClick={() => onDelete(server)}
        >
          <Trash2 className="h-4 w-4 text-critical" />
        </Button>
      </TableCell>
    </TableRow>
  );
}

export function AdminClientTable({ servers }: { servers: Server[] | undefined }) {
  const qc = useQueryClient();
  const [items, setItems] = useState<Server[]>([]);

  // Reconcile: preserve the user's drag order across live metric updates,
  // refreshing each row's data and adding/removing servers as they change.
  useEffect(() => {
    if (!servers) return;
    setItems((prev) => {
      if (prev.length === 0) return servers;
      const byId = new Map(servers.map((s) => [s.id, s]));
      const kept = prev.filter((p) => byId.has(p.id)).map((p) => byId.get(p.id)!);
      const keptIds = new Set(kept.map((k) => k.id));
      const added = servers.filter((s) => !keptIds.has(s.id));
      return [...kept, ...added];
    });
  }, [servers]);

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 5 } }),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates }),
  );

  const del = useMutation({
    mutationFn: (id: string) => deleteServer(id),
    onSuccess: () => {
      toast.success("Server removed");
      qc.invalidateQueries({ queryKey: serversKey });
    },
    onError: (e: Error) => toast.error(e.message || "Failed to delete"),
  });

  // Persist order: list is sorted by order_index DESC, so the top row gets the
  // highest index. Only changed rows are written.
  const persistOrder = async (ordered: Server[]) => {
    const writes = ordered
      .map((s, i) => ({ id: s.id, value: ordered.length - 1 - i, old: s.order_index }))
      .filter((w) => w.value !== w.old);
    if (writes.length === 0) return;
    try {
      await Promise.all(writes.map((w) => setServerOrder(w.id, w.value)));
      toast.success("Order updated");
    } catch {
      toast.error("Failed to save order");
    } finally {
      qc.invalidateQueries({ queryKey: serversKey });
    }
  };

  const onDragEnd = (event: DragEndEvent) => {
    const { active, over } = event;
    if (!over || active.id === over.id) return;
    const oldIndex = items.findIndex((s) => s.id === active.id);
    const newIndex = items.findIndex((s) => s.id === over.id);
    if (oldIndex < 0 || newIndex < 0) return;
    const next = arrayMove(items, oldIndex, newIndex);
    setItems(next.map((s, i) => ({ ...s, order_index: next.length - 1 - i })));
    void persistOrder(next);
  };

  if (items.length === 0) {
    return <EmptyState Icon={ServerOff} title="No clients yet" description="Register a client to get an install command." />;
  }

  return (
    <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={onDragEnd}>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead className="w-8" />
            <TableHead>Name</TableHead>
            <TableHead>Status</TableHead>
            <TableHead>Install command</TableHead>
            <TableHead className="w-16 text-right">Actions</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          <SortableContext items={items.map((s) => s.id)} strategy={verticalListSortingStrategy}>
            {items.map((s) => (
              <SortableRow
                key={s.id}
                server={s}
                onDelete={(srv) => {
                  if (window.confirm(`Delete ${srv.name}? This removes the server and its allow-list entry.`)) {
                    del.mutate(srv.id);
                  }
                }}
              />
            ))}
          </SortableContext>
        </TableBody>
      </Table>
      <p className="px-4 pb-3 pt-1 text-xs text-muted-foreground">Drag the handle to reorder how servers appear on the dashboard.</p>
    </DndContext>
  );
}

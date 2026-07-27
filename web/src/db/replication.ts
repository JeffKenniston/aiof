import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { OrchestratorService } from "../gen/proto/orchestrator/v1/orchestrator_connect";
import { Document } from "../gen/proto/orchestrator/v1/orchestrator_pb";
import { getDatabase } from "./index";

import { BACKEND_URL } from "../config";

// HTTP/3 (HTTP/2 cleartext backed) transport to the Go middleware
const transport = createConnectTransport({
  baseUrl: BACKEND_URL,
});

export const client = createClient(OrchestratorService, transport);

// Phase 3.1: pullHandler for zero-latency offline recovery
export async function pullHandler(sinceUpdatedAt: number, limit: number): Promise<Document[]> {
    const res = await client.pull({ sinceUpdatedAt: BigInt(sinceUpdatedAt), limit });
    return res.documents;
}

// Phase 3.1: pushHandler for synchronous WAL mutation resolving
export async function pushHandler(documents: Document[]): Promise<boolean> {
    const res = await client.push({ documents });
    return res.success;
}

// Phase 3.1: pullStream endpoint for zero-latency streaming multiplexing
export async function pullStream(getSince: () => number, signal: AbortSignal, onDoc: (doc: Document) => void) {
    let delay = 1000;
    
    while (!signal.aborted) {
        try {
            const currentSince = getSince();
            for await (const res of client.pullStream({ sinceUpdatedAt: BigInt(currentSince) }, { signal })) {
                if (res.document) {
                    onDoc(res.document);
                }
                delay = 1000; // Reset delay on successful message reception
            }
            // If the server cleanly ends the stream, wait before reconnecting
            if (!signal.aborted) {
                await new Promise(r => setTimeout(r, delay));
                delay = Math.min(delay * 1.5, 30000);
            }
        } catch (err: any) {
            if (err.name === 'AbortError' || signal.aborted) {
                break;
            }
            console.error('pullStream error (reconnecting):', err);
            await new Promise(r => setTimeout(r, delay));
            delay = Math.min(delay * 1.5, 30000);
        }
    }
}

// Phase 3.1: Start background replication
export async function startReplication(projectId: string, signal: AbortSignal) {
    const db = await getDatabase(projectId);
    
    let lastUpdatedAt = 0;
    // Initialize checkpoint from local DB
    try {
        const collections = ['telemetry', 'agent_logs', 'wal_events', 'agent_graph_state', 'sandbox_state', 'context_cache', 'task_queue', 'chat_messages'];
        for (const colName of collections) {
            const col = db[colName as keyof typeof db] as any;
            if (col) {
                const docs = await col.find({
                    sort: [{ updatedAt: 'desc' }],
                    limit: 1
                }).exec();
                if (docs.length > 0 && docs[0].updatedAt > lastUpdatedAt) {
                    lastUpdatedAt = docs[0].updatedAt;
                }
            }
        }
    } catch (e) {
        console.warn('[Replication] Failed to restore checkpoint, starting from zero', e);
    }

    pullStream(() => lastUpdatedAt, signal, async (doc) => {
        try {
            // ADR-04: Two-pass deserialization. Payload is raw bytes (JSON)
            const payload = JSON.parse(new TextDecoder().decode(doc.payload));
            
            const collectionMap: Record<string, string> = {
                'telemetry': 'telemetry',
                'agent_log': 'agent_logs',
                'wal_event': 'wal_events',
                'agent_graph_state': 'agent_graph_state',
                'sandbox_state': 'sandbox_state',
                'context_cache': 'context_cache',
                'task_queue': 'task_queue',
                'chat_messages': 'chat_messages'
            };
            const colName = collectionMap[doc.documentType];
            
            if (!colName) {
                console.log(`Unknown document type: ${doc.documentType}`);
                return;
            }
            
            const col = db[colName as keyof typeof db] as any;
            if (col) {
                if (doc.isDeleted) {
                    await col.findOne(doc.id).remove();
                } else {
                    await col.upsert({
                        ...payload,
                        id: doc.id,
                        updatedAt: Number(doc.updatedAt)
                    });
                }
                
                // Broadcast DB update so UI components (like FileExplorer) can react
                window.dispatchEvent(new CustomEvent('db-update', { 
                    detail: { collection: colName, doc: payload } 
                }));
            }
            
            if (Number(doc.updatedAt) > lastUpdatedAt) {
                lastUpdatedAt = Number(doc.updatedAt);
            }
        } catch (err) {
            console.error('Failed to apply replicated doc:', err);
        }
    }).catch(console.error);
}

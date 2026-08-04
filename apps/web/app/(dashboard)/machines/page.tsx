import { PageHeader } from "@/components/page-header";
import { tempoFetch } from "@/lib/api/client";
import type { Machine } from "@/lib/api/types";

export const dynamic="force-dynamic";
export default async function MachinesPage(){const {machines}=await tempoFetch<{machines:Machine[]}>("/api/v1/machines");return <><PageHeader title="Machines" subtitle="Authorized agents and last sync"/><div className="table-wrap">{machines.length?<table><thead><tr><th>Name</th><th>ID</th><th>Created</th><th>Last seen</th><th>Status</th></tr></thead><tbody>{machines.map(machine=>{const online=machine.last_seen_at&&Date.now()-new Date(machine.last_seen_at).getTime()<120000;return <tr key={machine.id}><td><strong>{machine.name}</strong></td><td>{machine.id.slice(0,13)}...</td><td>{date(machine.created_at)}</td><td>{machine.last_seen_at?date(machine.last_seen_at):"Never"}</td><td><span className={`badge ${online?"running":""}`}>{online?"Online":"Offline"}</span></td></tr>})}</tbody></table>:<div className="empty">No machines have been registered yet.</div>}</div></>}
function date(value:string){return new Intl.DateTimeFormat("en-GB",{dateStyle:"medium",timeStyle:"short"}).format(new Date(value))}

# Codex Tempo
## Plan de implementación actualizado

**Estado:** propuesta técnica
**Arquitectura objetivo:** Go + SQLite + PostgreSQL + Next.js
**Ámbito inicial:** single-user, self-hosted, orientado a homelab
**Objetivo principal:** contabilizar trabajo por proyecto sin perder el tiempo de ejecuciones paralelas

---

# 1. Resumen ejecutivo

Codex Tempo será una herramienta self-hosted para medir el trabajo realizado con agentes de programación, especialmente Codex CLI, cuando varias sesiones o proyectos se ejecutan al mismo tiempo.

El sistema no utilizará el modelo clásico de WakaTime basado en una única secuencia global de heartbeats. Cada sesión de agente tendrá su propia línea temporal y cada prompt o turno abrirá una ejecución independiente.

Ejemplo:

```text
Proyecto A: 10:00–10:20
Proyecto B: 10:05–10:25
```

Resultado esperado:

```text
Proyecto A             20 min de agente
Proyecto B             20 min de agente
Tiempo acumulado       40 min
Tiempo real            25 min
Paralelismo máximo      2
```

La arquitectura tendrá tres aplicaciones:

1. **Agente local en Go**
   - Observa hooks y transcripciones de Codex.
   - Mantiene cursores independientes por sesión.
   - Guarda eventos en SQLite.
   - Funciona sin conexión.
   - Sincroniza por lotes con el servidor.

2. **API y motor de dominio en Go**
   - Recibe eventos idempotentes.
   - Proyecta sesiones y ejecuciones.
   - Calcula intervalos y métricas.
   - Expone API REST y eventos en tiempo real.
   - Usa PostgreSQL como fuente de verdad.

3. **Dashboard en Next.js**
   - Presenta informes, timelines y sesiones activas.
   - Consume un cliente TypeScript generado desde OpenAPI.
   - Mantiene la lógica temporal en el backend.
   - Actúa como BFF para evitar exponer secretos al navegador.

---

# 2. Objetivos

## 2.1 Objetivos principales

- Contabilizar tiempo por proyecto aunque existan varios proyectos activos en paralelo.
- Separar tiempo de agente, tiempo real y tiempo humano.
- Identificar proyecto, máquina, sesión y ejecución de forma estable.
- Evitar pérdidas por cursores temporales globales.
- Tolerar reinicios, duplicados, escrituras tardías y eventos fuera de orden.
- Funcionar localmente sin conexión al servidor.
- Poder reconstruir todas las proyecciones a partir de eventos inmutables.
- Ofrecer una experiencia visual adecuada para timelines complejas.
- Mantener el despliegue razonable para un homelab.

## 2.2 Métricas principales

| Métrica | Definición |
|---|---|
| `agent_time` | Suma de la duración de todas las ejecuciones. Los solapamientos cuentan. |
| `project_span` | Unión de intervalos dentro de un proyecto. Los solapamientos internos no se duplican. |
| `wall_clock` | Unión global de todos los intervalos. |
| `human_time` | Ventanas de interacción humana activa, implementadas después del MVP. |
| `parallelism_peak` | Máximo número de ejecuciones simultáneas. |
| `parallelism_average` | Paralelismo medio ponderado durante el periodo. |
| `run_count` | Número de ejecuciones iniciadas en el periodo. |
| `token_usage` | Tokens de entrada y salida por proyecto, sesión y modelo. |

## 2.3 No objetivos iniciales

- Reemplazar Wakapi como contador de edición manual en IDE.
- Guardar prompts, respuestas o código por defecto.
- Crear un producto multiempresa.
- Implementar facturación.
- Soportar todos los agentes desde la primera versión.
- Ejecutar análisis semántico del contenido.
- Escalar horizontalmente desde el primer día.
- Crear una aplicación móvil.

---

# 3. Stack recomendado

# 3.1 Agente local

```text
Go
SQLite WAL
modernc.org/sqlite
fsnotify
TOML
systemd / launchd / Windows service
```

## Razones

- Binario único.
- Bajo consumo.
- Buena concurrencia.
- Fácil distribución multiplataforma.
- Acceso directo al sistema de archivos y Git.
- Buen soporte para servicios residentes.
- Posibilidad de compartir dominio y tipos con el backend.

## Librerías sugeridas

- `modernc.org/sqlite`: SQLite sin CGO.
- `github.com/fsnotify/fsnotify`: observación de archivos.
- `github.com/BurntSushi/toml` o equivalente: configuración.
- `github.com/google/uuid` o UUIDv7 compatible.
- `log/slog`: logging estructurado.

# 3.2 Backend

```text
Go
net/http
chi
PostgreSQL
pgx
sqlc
goose
OpenAPI
SSE
Prometheus
```

## Razones

El backend es responsable del comportamiento correcto del sistema. Toda la lógica de:

- apertura y cierre de runs;
- deduplicación;
- unión de intervalos;
- métricas;
- reconstrucción de proyecciones;
- permisos;

debe estar en Go, no en el frontend.

## Librerías sugeridas

- `github.com/go-chi/chi/v5`: routing HTTP.
- `github.com/jackc/pgx/v5`: PostgreSQL.
- `sqlc`: código tipado desde SQL.
- `goose`: migraciones.
- `kin-openapi` o generación equivalente.
- Cliente Prometheus para Go.
- `log/slog`.

# 3.3 Frontend

```text
Next.js App Router
TypeScript
React
Tailwind CSS
shadcn/ui
TanStack Query
Zod
React Hook Form
Apache ECharts
OpenAPI-generated client
```

## Razones

Next.js encaja mejor que HTMX para:

- timelines con carriles;
- zoom;
- filtros interactivos;
- comparaciones temporales;
- sesiones activas;
- tooltips;
- actualización en tiempo real;
- edición manual de runs;
- navegación entre proyectos, sesiones y ejecuciones.

## Principio de diseño

```text
Go calcula.
PostgreSQL almacena.
Next.js presenta.
```

El frontend no debe volver a implementar cálculos temporales.

# 3.4 Persistencia

## SQLite local

Almacena:

- cola de eventos;
- cursores;
- estado de sincronización;
- configuración local;
- proyectos detectados;
- sesiones abiertas;
- errores de parser;
- eventos confirmados durante un periodo de retención.

Configuración recomendada:

```sql
PRAGMA journal_mode = WAL;
PRAGMA foreign_keys = ON;
PRAGMA synchronous = NORMAL;
PRAGMA busy_timeout = 5000;
```

## PostgreSQL

Almacena:

- máquinas;
- proyectos;
- sesiones;
- eventos originales;
- runs proyectados;
- rollups;
- aliases;
- configuración;
- auditoría.

---

# 4. Arquitectura

```text
┌──────────────────────────────────────────────────────────────┐
│ Máquina de desarrollo                                       │
│                                                              │
│  Codex hooks ───────┐                                       │
│                     ├──> codex-tempo-agent                  │
│  JSONL transcripts ─┘       │                                │
│                             ├── parser                       │
│                             ├── state machine                │
│                             ├── project resolver             │
│                             ├── SQLite WAL                   │
│                             ├── local CLI                    │
│                             └── sync worker                  │
└──────────────────────────────────┬───────────────────────────┘
                                   │ HTTPS
                                   │ batch events
                                   ▼
┌──────────────────────────────────────────────────────────────┐
│ Homelab                                                      │
│                                                              │
│  reverse proxy                                               │
│       │                                                      │
│       ├──────────────> Next.js                               │
│       │                    │                                 │
│       │                    │ internal API                    │
│       │                    ▼                                 │
│       └──────────────> Go API                                │
│                            │                                 │
│                            ├── ingest                        │
│                            ├── projector                     │
│                            ├── reports                       │
│                            ├── SSE                           │
│                            └── PostgreSQL                    │
└──────────────────────────────────────────────────────────────┘
```

---

# 5. Principios arquitectónicos

1. **Event sourcing limitado**
   - Los eventos originales son inmutables.
   - Las proyecciones pueden reconstruirse.
   - No es necesario aplicar event sourcing a toda la configuración.

2. **Estado independiente por sesión**
   - Nunca existe un único `latestHeartbeat`.
   - La estructura fundamental es `stateBySession`.

3. **Idempotencia end-to-end**
   - Agente, API y proyector toleran reintentos.

4. **Orden parcial**
   - Existe orden por sesión.
   - No existe un orden global fiable entre máquinas o sesiones.

5. **Offline first**
   - El agente siempre escribe localmente antes de enviar.

6. **Privacidad por defecto**
   - El contenido no se almacena salvo activación explícita.

7. **Frontend sin lógica de dominio**
   - El navegador representa datos ya calculados.

8. **Proyecciones versionadas**
   - Cada run y agregado registra la versión del algoritmo.

---

# 6. Modelo de dominio

# 6.1 Machine

Representa una instalación del agente.

```go
type Machine struct {
    ID         string
    Name       string
    CreatedAt  time.Time
    LastSeenAt *time.Time
}
```

# 6.2 Project

```go
type Project struct {
    ID          string
    Fingerprint string
    Name        string
    RootPath    string
    RemoteHash  string
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

El fingerprint debe ser estable aunque cambie el nombre del directorio.

Orden de resolución:

1. Remote Git normalizado.
2. Raíz Git.
3. Worktree.
4. Ruta real.
5. `cwd`.

# 6.3 Session

Una instancia interactiva de Codex.

```go
type Session struct {
    ID           string
    MachineID    string
    ProjectID    string
    CWD          string
    Source       string
    CodexVersion string
    StartedAt    time.Time
    EndedAt      *time.Time
}
```

# 6.4 Run

Un turno iniciado por un prompt.

```go
type Run struct {
    ID                string
    SessionID         string
    ProjectID         string
    StartedAt         time.Time
    EndedAt           *time.Time
    Status            RunStatus
    Model             string
    ReasoningEffort   string
    InputTokens       int64
    CachedInputTokens int64
    OutputTokens      int64
    ReasoningTokens   int64
    CloseReason       string
    ProjectionVersion int
}
```

# 6.5 Event

```go
type Event struct {
    ID         string
    MachineID  string
    SessionID  string
    RunID      string
    Sequence   int64
    OccurredAt time.Time
    Kind       EventKind
    Source     EventSource
    Payload    json.RawMessage
}
```

# 6.6 TranscriptCursor

```go
type TranscriptCursor struct {
    Path         string
    FileIdentity string
    ByteOffset   int64
    LastEventID  string
    LastSequence int64
    LastModified time.Time
}
```

---

# 7. Identificadores

## 7.1 `machine_id`

UUID generado al instalar el agente.

## 7.2 `session_id`

Usar el ID nativo de Codex.

Fallback:

```text
sha256(machine_id + transcript_path + first_event_timestamp)
```

## 7.3 `run_id`

Preferencia:

1. ID nativo de turno.
2. UUIDv7 generado al observar el prompt.
3. ID determinista durante reparsing.

## 7.4 `event_id`

Debe deduplicar la misma señal recibida por hook y transcripción.

Preferencia:

```text
sha256(
  machine_id +
  session_id +
  source_event_type +
  native_event_id
)
```

Fallback:

```text
sha256(
  session_id +
  timestamp_nanos +
  canonical_payload
)
```

## 7.5 `project_id`

```text
project_fingerprint = sha256(
  normalized_remote +
  repository_subpath
)
```

Para proyectos sin Git:

```text
sha256(machine_id + canonical_realpath)
```

---

# 8. Eventos

Conjunto inicial:

```text
session_started
session_closed
prompt_submitted
run_started
tool_started
tool_finished
assistant_message
run_completed
run_interrupted
run_failed
lease
token_usage
project_changed
```

## Fuentes

- Hook de Codex.
- Transcripción JSONL.
- Inferencia del agente.
- Reparación manual.
- API administrativa.

Cada evento debe registrar `source`.

---

# 9. Máquina de estados

```text
IDLE
  │
  └── prompt_submitted / run_started
          ▼
       RUNNING
          │
          ├── tool_started
          ├── tool_finished
          ├── assistant_message
          ├── token_usage
          │      └── permanece RUNNING
          │
          ├── run_completed ───────> COMPLETED
          ├── run_interrupted ─────> INTERRUPTED
          ├── run_failed ──────────> FAILED
          ├── session_closed ──────> INTERRUPTED
          ├── nuevo run ───────────> SUPERSEDED
          └── lease vencido ───────> ABANDONED
```

## Reglas

- Una sesión solo tiene un run activo inicialmente.
- Otra sesión nunca modifica este estado.
- Otro proyecto nunca modifica este estado.
- Un evento tardío puede corregir un cierre inferido.
- Los cambios deben ser deterministas.

## Orden de cierre

1. Finalización explícita.
2. Interrupción.
3. Fallo.
4. Cierre de sesión.
5. Nuevo run en la misma sesión.
6. Lease vencido.
7. Timeout máximo.

---

# 10. Leases y procesos muertos

Un run puede quedar abierto si:

- se cierra la terminal;
- se apaga la máquina;
- se mata Codex;
- no aparece un evento final.

Solución:

- Emitir o inferir lease periódicamente.
- Guardar `last_activity_at`.
- Cerrar con `ABANDONED`.
- Permitir que un evento tardío sustituya el cierre.

Configuración inicial sugerida:

```text
lease_interval: 30s
lease_grace_period: 90s
maximum_open_run: 12h
```

Estos valores deben validarse con datos reales.

---

# 11. Esquema PostgreSQL

```sql
CREATE TABLE machines (
    id uuid PRIMARY KEY,
    name text NOT NULL,
    token_hash bytea NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    last_seen_at timestamptz
);

CREATE TABLE projects (
    id uuid PRIMARY KEY,
    fingerprint text NOT NULL UNIQUE,
    name text NOT NULL,
    remote_hash text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE sessions (
    id text PRIMARY KEY,
    machine_id uuid NOT NULL REFERENCES machines(id),
    project_id uuid NOT NULL REFERENCES projects(id),
    cwd text,
    source text NOT NULL,
    codex_version text,
    started_at timestamptz NOT NULL,
    ended_at timestamptz
);

CREATE TABLE events (
    id uuid PRIMARY KEY,
    machine_id uuid NOT NULL REFERENCES machines(id),
    session_id text NOT NULL,
    run_id uuid,
    sequence bigint NOT NULL,
    occurred_at timestamptz NOT NULL,
    kind text NOT NULL,
    source text NOT NULL,
    payload jsonb NOT NULL DEFAULT '{}'::jsonb,
    received_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (machine_id, session_id, sequence)
);

CREATE INDEX events_session_order_idx
    ON events (session_id, occurred_at, sequence);

CREATE INDEX events_time_idx
    ON events (occurred_at);

CREATE TABLE runs (
    id uuid PRIMARY KEY,
    session_id text NOT NULL REFERENCES sessions(id),
    project_id uuid NOT NULL REFERENCES projects(id),
    started_at timestamptz NOT NULL,
    ended_at timestamptz,
    status text NOT NULL,
    model text,
    reasoning_effort text,
    input_tokens bigint NOT NULL DEFAULT 0,
    cached_input_tokens bigint NOT NULL DEFAULT 0,
    output_tokens bigint NOT NULL DEFAULT 0,
    reasoning_tokens bigint NOT NULL DEFAULT 0,
    close_reason text,
    projection_version integer NOT NULL DEFAULT 1,
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (ended_at IS NULL OR ended_at >= started_at)
);

CREATE INDEX runs_project_time_idx
    ON runs (project_id, started_at, ended_at);

CREATE INDEX runs_active_idx
    ON runs (session_id)
    WHERE ended_at IS NULL;

CREATE TABLE project_aliases (
    project_id uuid NOT NULL REFERENCES projects(id),
    alias text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (project_id, alias)
);

CREATE TABLE projection_checkpoints (
    projector text PRIMARY KEY,
    last_event_received_at timestamptz,
    projection_version integer NOT NULL
);
```

## Tablas posteriores

- `daily_project_rollups`.
- `human_interactions`.
- `project_labels`.
- `manual_run_adjustments`.
- `audit_log`.
- `users`.
- `sessions_auth`.

---

# 12. Cálculo de métricas

# 12.1 Semántica de intervalos

Usar intervalos:

```text
[start, end)
```

El final no pertenece al intervalo.

## `agent_time`

Suma todos los runs recortados al periodo.

## `project_span`

Une intervalos por proyecto.

## `wall_clock`

Une todos los intervalos.

## Paralelismo

Crear puntos:

```text
(start, +1)
(end, -1)
```

Cuando dos puntos comparten timestamp:

1. procesar finales;
2. procesar inicios.

## Invariantes

- `wall_clock <= agent_time`.
- `project_span <= agent_time` por proyecto.
- Ninguna duración es negativa.
- Reprocesar eventos produce el mismo resultado.
- Duplicar eventos no cambia métricas.

---

# 13. API

# 13.1 Separación de APIs

## API de ingestión

Consumida por agentes.

```text
POST /api/v1/ingest/events
POST /api/v1/ingest/register
GET  /api/v1/ingest/sync-state
```

## API de producto

Consumida por Next.js.

```text
GET /api/v1/projects
GET /api/v1/projects/{id}
GET /api/v1/reports/summary
GET /api/v1/reports/timeline
GET /api/v1/runs
GET /api/v1/runs/active
GET /api/v1/sessions
GET /api/v1/machines
GET /api/v1/events/stream
```

## API administrativa

```text
PATCH /api/v1/projects/{id}
POST  /api/v1/projects/{id}/aliases
PATCH /api/v1/runs/{id}
POST  /api/v1/admin/rebuild
POST  /api/v1/machines/{id}/rotate-token
DELETE /api/v1/machines/{id}
```

# 13.2 Ingestión por lotes

```http
POST /api/v1/ingest/events
Authorization: Bearer <machine-token>
Content-Type: application/json
```

```json
{
  "machine_id": "uuid",
  "events": [
    {
      "id": "uuid",
      "session_id": "session-123",
      "run_id": "uuid",
      "sequence": 42,
      "occurred_at": "2026-08-04T00:10:00.123Z",
      "kind": "run_completed",
      "source": "transcript",
      "payload": {}
    }
  ]
}
```

Respuesta:

```json
{
  "accepted": 20,
  "duplicates": 3,
  "rejected": [
    {
      "id": "uuid",
      "reason": "invalid_timestamp"
    }
  ]
}
```

# 13.3 OpenAPI

El backend es la fuente de verdad del contrato.

Pipeline:

```text
OpenAPI
   ├── validación backend
   ├── documentación
   └── cliente TypeScript generado
```

El frontend no debe definir a mano los DTO principales.

---

# 14. BFF con Next.js

## Objetivo

Evitar que el navegador conozca:

- tokens internos;
- topología del backend;
- direcciones internas;
- credenciales de servicio.

## Flujo

```text
Navegador
   ↓ cookie de sesión
Next.js
   ↓ token interno
Go API
```

## Responsabilidades de Next.js

- Autenticación web.
- Renderizado inicial.
- Proxy de consultas.
- Server Actions para mutaciones.
- Validación superficial de formularios.
- Caching de presentación.
- Entrega de assets.

## Responsabilidades del backend Go

- Autorización real.
- Validación de negocio.
- Cálculos.
- Persistencia.
- Auditoría.
- Control de acceso a máquinas y proyectos.

---

# 15. Frontend Next.js

# 15.1 App Router

Usar:

- Server Components para páginas y carga inicial.
- Client Components para timelines, gráficos y filtros.
- Route Handlers solo cuando aporten valor como BFF.
- Server Actions para formularios sencillos.
- Renderizado dinámico en páginas temporales.

# 15.2 Estructura

```text
apps/web/
├── app/
│   ├── (dashboard)/
│   │   ├── page.tsx
│   │   ├── projects/
│   │   ├── timeline/
│   │   ├── runs/
│   │   ├── sessions/
│   │   ├── machines/
│   │   └── settings/
│   ├── api/
│   ├── login/
│   └── layout.tsx
├── components/
│   ├── charts/
│   ├── timeline/
│   ├── filters/
│   ├── tables/
│   └── ui/
├── lib/
│   ├── api/
│   ├── auth/
│   ├── dates/
│   └── validation/
├── hooks/
├── styles/
└── tests/
```

# 15.3 Pantallas

## Dashboard

- Tiempo de agente.
- Tiempo real.
- Paralelismo máximo.
- Número de runs.
- Tokens.
- Comparación con periodo anterior.
- Proyectos principales.
- Sesiones activas.

## Timeline

- Carril por sesión.
- Agrupación opcional por proyecto.
- Zoom.
- Pan horizontal.
- Tooltips.
- Eventos de inicio y cierre.
- Runs inferidos marcados visualmente.
- Selección de intervalo.
- Filtros.

## Proyecto

- Resumen.
- Runs.
- Sesiones.
- Modelos.
- Tokens.
- Tendencia diaria.
- Aliases.
- Edición de nombre.

## Runs activos

- Máquina.
- Proyecto.
- Sesión.
- Inicio.
- Duración actual.
- Última actividad.
- Modelo.
- Estado del lease.

## Diagnóstico

- Cola por máquina.
- Última sincronización.
- Errores de parser.
- Eventos rechazados.
- Proyección atrasada.

# 15.4 Estado y fetching

## Server Components

Para:

- carga inicial;
- páginas de informe;
- metadatos;
- navegación.

## TanStack Query

Para:

- runs activos;
- polling;
- timeline dinámica;
- filtros sin recarga;
- invalidación tras mutaciones.

No usar TanStack Query en todas las páginas por defecto.

# 15.5 Gráficos

Usar Apache ECharts para:

- tiempo por proyecto;
- tendencia;
- tokens;
- paralelismo;
- distribución por modelo.

La timeline puede empezar con ECharts custom series. Si limita la interacción, crear un componente SVG/canvas específico.

# 15.6 Fechas

Reglas:

- API transmite UTC.
- Frontend presenta zona configurada.
- Backend recorta intervalos.
- El navegador no calcula duración oficial.
- Usar una única librería de fechas en frontend.
- Incluir zona horaria explícita en filtros.

---

# 16. Tiempo real

## MVP

Polling cada 15–30 segundos para runs activos.

## Evolución

SSE:

```text
GET /api/v1/events/stream
```

Eventos:

```text
run.started
run.updated
run.closed
machine.online
machine.offline
projection.updated
```

Next.js puede:

- proxyar SSE;
- o permitir conexión directa a la API con cookie o token efímero.

Para el MVP, polling es suficiente.

---

# 17. Repositorio

```text
codex-tempo/
├── apps/
│   ├── agent/
│   │   └── main.go
│   ├── server/
│   │   └── main.go
│   ├── cli/
│   │   └── main.go
│   └── web/
│       ├── app/
│       ├── components/
│       ├── lib/
│       └── package.json
├── internal/
│   ├── agent/
│   ├── api/
│   ├── auth/
│   ├── codex/
│   ├── config/
│   ├── domain/
│   ├── interval/
│   ├── localdb/
│   ├── postgres/
│   ├── projector/
│   ├── projectresolver/
│   ├── reports/
│   └── sync/
├── db/
│   ├── migrations/
│   ├── queries/
│   └── sqlc.yaml
├── api/
│   ├── openapi.yaml
│   └── generated/
├── packages/
│   └── api-client/
├── deploy/
│   ├── docker/
│   ├── compose/
│   ├── systemd/
│   └── examples/
├── docs/
│   ├── architecture.md
│   ├── event-model.md
│   ├── privacy.md
│   └── adr/
├── testdata/
├── go.mod
├── package.json
├── pnpm-workspace.yaml
├── Taskfile.yml
└── README.md
```

## Decisiones

- Monorepo.
- Un módulo Go inicialmente.
- Workspace pnpm.
- Sin Turborepo en el MVP.
- Taskfile o Makefile como interfaz común.

---

# 18. Fases de implementación

# Fase 0: investigación

## Objetivo

Entender el formato real de Codex antes de fijar el parser.

## Trabajo

- Capturar sesiones simples.
- Capturar dos sesiones paralelas.
- Capturar dos proyectos paralelos.
- Capturar dos sesiones del mismo proyecto.
- Capturar interrupciones.
- Capturar cierres normales.
- Capturar fallos.
- Capturar cambios de modelo.
- Capturar token usage.
- Documentar hooks.
- Anonimizar fixtures.

## Entregables

- Fixtures JSONL.
- Documento del formato.
- Matriz evento-origen.
- ADR de fuente de verdad.

## Salida

Se puede reconocer:

- sesión;
- proyecto;
- prompt;
- inicio;
- final;
- tokens.

---

# Fase 1: dominio e intervalos

## Trabajo

- Tipos de dominio.
- IDs.
- Máquina de estados.
- Unión de intervalos.
- Recorte.
- Paralelismo.
- Tests.
- Proyección determinista.

## Criterios

- Dos sesiones paralelas no interfieren.
- Reprocesar produce el mismo resultado.
- Duplicar eventos no cambia el resultado.
- No existen duraciones negativas.

---

# Fase 2: agente local

## Trabajo

- Configuración.
- Identidad de máquina.
- SQLite.
- Cola durable.
- Cursores.
- Watchers.
- Escaneo de respaldo.
- Parser incremental.
- Hooks.
- Project resolver.
- Doctor.
- Reparse.

## Consideración

`fsnotify` no es recursivo. El agente debe:

- recorrer directorios;
- añadir watchers;
- detectar nuevos subdirectorios;
- hacer escaneo periódico.

## Criterio

Reiniciar el agente o perder red no pierde eventos.

---

# Fase 3: CLI e informes locales

## Comandos

```bash
codex-tempo report today
codex-tempo report week
codex-tempo timeline today
codex-tempo sessions --active
codex-tempo projects
codex-tempo doctor
codex-tempo reparse
```

## Criterio

El producto es útil sin servidor.

---

# Fase 4: backend

## Trabajo

- Migraciones.
- sqlc.
- Registro de máquina.
- Tokens.
- Ingestión batch.
- Idempotencia.
- Proyector.
- Rebuild.
- Informes.
- Métricas.
- Health checks.

## Criterio

Varias máquinas sincronizan sin duplicación.

---

# Fase 5: contrato OpenAPI

## Trabajo

- Especificación.
- Validación.
- Cliente TypeScript.
- CI.
- Documentación.
- Versionado.

## Criterio

Cambiar DTOs rompe CI si el frontend no se regenera.

---

# Fase 6: Next.js base

## Trabajo

- App Router.
- Autenticación.
- Layout.
- Cliente API.
- Dashboard inicial.
- Tabla de proyectos.
- Páginas de proyecto.
- Diseño responsive.
- Manejo de errores.

## Criterio

Dashboard y CLI muestran las mismas cifras.

---

# Fase 7: timeline interactiva

## Trabajo

- Carriles.
- Zoom.
- Pan.
- Tooltips.
- Selección de rango.
- Agrupación.
- Filtros.
- Runs inferidos.
- Rendimiento.

## Criterio

Se visualizan cientos o miles de runs sin bloquear la UI.

---

# Fase 8: sesiones activas

## Trabajo

- Polling.
- Estado de lease.
- Duración en vivo.
- Alertas de runs abandonados.
- Estado de máquinas.
- SSE opcional.

---

# Fase 9: edición y correcciones

## Trabajo

- Corregir inicio y final.
- Reasignar proyecto.
- Unir proyectos.
- Alias.
- Marcar run inválido.
- Auditoría.
- Rebuild seguro.

Las correcciones deben almacenarse como ajustes, no modificando eventos originales.

---

# Fase 10: tiempo humano

## Fuentes

- Prompt.
- Confirmaciones.
- Terminal.
- tmux.
- Wakapi.
- Ventana activa opcional.

## Política inicial

- Ventana de 90 segundos.
- Última interacción gana.
- No duplicar atención humana.
- Mostrar heurística.

---

# Fase 11: endurecimiento

## Trabajo

- Instaladores.
- Releases.
- Firma.
- SBOM.
- Backups.
- Restore.
- CI multiplataforma.
- Rate limiting.
- Retención.
- Prometheus.
- Documentación.

---

# 19. Estrategia de sincronización

Estados locales:

```text
pending
in_flight
acknowledged
rejected
```

## Reglas

- Enviar lotes limitados por eventos y bytes.
- Backoff exponencial con jitter.
- Aceptación parcial.
- Mantener eventos confirmados durante un periodo.
- Reintentar duplicados sin peligro.
- Validar reloj.
- Registrar desfase por máquina.

## Reconciliación

El servidor devuelve:

- último evento conocido;
- secuencia máxima por sesión;
- rechazos;
- versión de protocolo.

---

# 20. Autenticación y seguridad

# 20.1 Agentes

- Token por máquina.
- Alta entropía.
- Hash en servidor.
- Rotación.
- Revocación.
- TLS.
- Permisos limitados.

# 20.2 Web

Para single-user:

- autenticación local;
- OIDC opcional;
- cookie HTTP-only;
- CSRF;
- SameSite;
- sesión corta renovable.

# 20.3 Red

- API interna no expuesta salvo ingestión.
- Reverse proxy.
- Cabeceras de seguridad.
- Límites de payload.
- Timeouts.
- Rate limiting.

---

# 21. Privacidad

## No almacenar por defecto

- prompts;
- respuestas;
- código;
- diffs;
- contenido de archivos;
- argumentos completos;
- remotes en texto claro;
- variables de entorno.

## Almacenar

- IDs;
- timestamps;
- proyecto;
- sesión;
- modelo;
- estado;
- tokens;
- tipo de herramienta;
- razón de cierre.

## Configuración opcional

```toml
[privacy]
store_paths = false
store_tool_names = true
store_prompt_metadata = false
store_content = false
```

---

# 22. Observabilidad

## Backend

- `/healthz`.
- `/readyz`.
- `/metrics`.
- logs JSON.
- request ID.
- latencia.
- errores.
- proyección atrasada.

## Agente

- tamaño de cola;
- último sync;
- errores de parser;
- cursores;
- sesiones activas;
- estado de watcher.

## Frontend

- errores de renderizado;
- errores API;
- Web Vitals opcionales;
- sin analítica externa por defecto.

---

# 23. Pruebas

# 23.1 Go

- unitarias;
- integración;
- race detector;
- propiedades;
- fixtures;
- PostgreSQL real;
- SQLite real.

# 23.2 Frontend

- Vitest.
- Testing Library.
- Playwright.
- Tests visuales opcionales.

# 23.3 Escenarios obligatorios

1. Dos proyectos paralelos.
2. Dos sesiones del mismo proyecto.
3. Run sin tool calls.
4. Ctrl+C.
5. Apagado abrupto.
6. Eventos fuera de orden.
7. Evento tardío.
8. Duplicado hook/transcript.
9. Cambio de cwd.
10. Worktree.
11. Proyecto sin Git.
12. Reloj desfasado.
13. Rebuild completo.
14. Corrección manual.
15. Timeline con gran volumen.

---

# 24. CI/CD

## Backend y agente

1. fmt.
2. vet.
3. lint.
4. tests.
5. race.
6. integración PostgreSQL.
7. migraciones.
8. sqlc.
9. build multiplataforma.

## Frontend

1. lint.
2. typecheck.
3. tests.
4. build.
5. Playwright.
6. cliente OpenAPI actualizado.

## Release

- binarios;
- checksums;
- firmas;
- SBOM;
- imágenes multiarch;
- changelog;
- migraciones documentadas.

---

# 25. Docker Compose

```yaml
services:
  api:
    image: ghcr.io/example/codex-tempo-api:latest
    restart: unless-stopped
    environment:
      DATABASE_URL: postgres://tempo:${POSTGRES_PASSWORD}@postgres:5432/tempo
      LISTEN_ADDR: 0.0.0.0:8080
    depends_on:
      postgres:
        condition: service_healthy

  web:
    image: ghcr.io/example/codex-tempo-web:latest
    restart: unless-stopped
    environment:
      INTERNAL_API_URL: http://api:8080
      INTERNAL_API_TOKEN: ${INTERNAL_API_TOKEN}
    depends_on:
      - api

  postgres:
    image: postgres:18
    restart: unless-stopped
    environment:
      POSTGRES_DB: tempo
      POSTGRES_USER: tempo
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U tempo -d tempo"]
      interval: 10s
      timeout: 5s
      retries: 5
    volumes:
      - postgres-data:/var/lib/postgresql/data

volumes:
  postgres-data:
```

---

# 26. Despliegue del agente

## Linux

```ini
[Unit]
Description=Codex Tempo Agent
After=network-online.target

[Service]
ExecStart=%h/.local/bin/codex-tempo-agent run
Restart=on-failure
RestartSec=5

[Install]
WantedBy=default.target
```

## Directorios

```text
~/.config/codex-tempo/config.toml
~/.local/share/codex-tempo/tempo.db
~/.local/state/codex-tempo/agent.log
```

---

# 27. Backup y recuperación

## PostgreSQL

- dump diario;
- retención;
- copia fuera del host;
- prueba de restore;
- backup antes de migración.

## Eventos

Exportación JSONL:

```bash
codex-tempo export events --output events.jsonl
```

## Reconstrucción

```bash
codex-tempo admin rebuild --projection-version latest
```

---

# 28. Riesgos

| Riesgo | Impacto | Mitigación |
|---|---|---|
| Cambia el JSONL de Codex | Alto | Parser versionado y fixtures. |
| No hay evento final | Alto | Lease y timeout. |
| Evento tardío | Alto | Offset por archivo y reconciliación. |
| Duplicado hook/transcript | Medio | Event ID determinista. |
| `fsnotify` pierde cambios | Medio | Escaneo periódico. |
| Proyecto duplicado | Medio | Fingerprint y merge. |
| Next.js aumenta complejidad | Medio | BFF pequeño y lógica en Go. |
| Contrato frontend/backend diverge | Medio | OpenAPI generado y CI. |
| Timeline lenta | Medio | Agregación, virtualización y canvas. |
| Runs abiertos eternamente | Medio | Reconciliación. |
| Métricas mal interpretadas | Alto | Nombres y definiciones visibles. |
| Datos sensibles | Alto | Privacidad por defecto. |

---

# 29. Criterios de aceptación del MVP

El MVP se considera completo cuando:

- Detecta dos sesiones paralelas.
- Identifica el proyecto correcto.
- Mantiene cursor por transcripción.
- Abre y cierra runs por sesión.
- Funciona offline.
- Sincroniza sin duplicar.
- Calcula `agent_time`.
- Calcula `project_span`.
- Calcula `wall_clock`.
- Calcula paralelismo.
- Tiene CLI local.
- Tiene dashboard Next.js.
- La timeline muestra solapamientos.
- CLI y web coinciden.
- Permite reparse.
- No guarda prompts.
- Incluye tests del caso principal.

Caso principal:

```text
A: 10:00–10:20
B: 10:05–10:25
```

Resultado:

```text
A agent_time:     20 min
B agent_time:     20 min
Total agent_time: 40 min
Wall clock:       25 min
Peak parallelism: 2
```

---

# 30. Backlog inicial

1. Crear monorepo.
2. Configurar Go y pnpm.
3. Configurar CI.
4. Capturar fixtures.
5. Documentar JSONL.
6. Definir eventos.
7. Implementar IDs.
8. Implementar máquina de estados.
9. Implementar intervalos.
10. Implementar SQLite.
11. Implementar cursores.
12. Implementar parser.
13. Implementar project resolver.
14. Implementar CLI.
15. Implementar PostgreSQL.
16. Implementar ingestión.
17. Implementar proyector.
18. Implementar informes.
19. Escribir OpenAPI.
20. Generar cliente TS.
21. Crear Next.js.
22. Crear dashboard.
23. Crear timeline.
24. Crear runs activos.
25. Añadir edición.
26. Añadir Prometheus.
27. Crear instaladores.
28. Documentar despliegue.

---

# 31. Orden recomendado de desarrollo

```text
Fixtures
  ↓
Dominio
  ↓
Parser
  ↓
SQLite
  ↓
CLI local
  ↓
API
  ↓
PostgreSQL
  ↓
OpenAPI
  ↓
Next.js
  ↓
Timeline
  ↓
Sincronización en vivo
  ↓
Tiempo humano
```

No comenzar por el dashboard. Primero deben ser correctos los eventos, los intervalos y las métricas.

---

# 32. Decisiones pendientes

1. ¿Codex ofrece un ID de turno estable?
2. ¿Qué evento representa el final real?
3. ¿Dos agentes del mismo proyecto suman ambos?
4. ¿Cuánto tiempo se retienen eventos locales?
5. ¿Se almacenan rutas?
6. ¿Se quiere importar Wakapi?
7. ¿Se añadirá Claude Code?
8. ¿OIDC desde el MVP?
9. ¿La API de ingestión estará expuesta a Internet?
10. ¿Se necesita multiusuario?
11. ¿Se permite corregir runs desde la web?
12. ¿Qué timeout de lease funciona mejor?

## Recomendación inicial

- Sumar agentes paralelos del mismo proyecto.
- Mostrar también unión por proyecto.
- Single-user.
- OIDC opcional.
- Correcciones manuales auditadas.
- Contenido desactivado.
- API de ingestión detrás de TLS.
- Arquitectura preparada para parsers adicionales.

---

# 33. Resultado técnico recomendado

```text
Agente:
  Go + SQLite + fsnotify

Backend:
  Go + chi + pgx + sqlc + goose + PostgreSQL

Contrato:
  OpenAPI + cliente TypeScript generado

Frontend:
  Next.js App Router + TypeScript
  Tailwind + shadcn/ui
  TanStack Query
  ECharts

Despliegue:
  agente nativo
  API, web y PostgreSQL en Docker Compose
```

Esta arquitectura mantiene la precisión temporal en Go y utiliza Next.js donde aporta más valor: visualización, interacción y experiencia de usuario.

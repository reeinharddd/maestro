# Skills & Context Problem — Handoff para Maestro

> Descubierto: 2026-06-30
> Próximo paso: Implementar en maestro `skill` subsystem

## El Problema

opencode inyecta **~130KB de system prompt** en cada sesión. Un simple "hola" gasta 140K+ tokens. El desglose:

| Componente | Tamaño | Detalle |
|---|---|---|
| Behavior_Instructions | ~35KB | Fases 0-3, tool rules, delegación |
| `<available_skills>` XML | **~28KB** | **205 skills** con nombre+descripción+location |
| `<available_commands>` | ~15KB | 104 comandos |
| AGENTS.md | 13KB | Constitution + protocolo |
| 78 subagentes | ~10KB | Prompts de agentes en opencode.json |
| Tool schemas | ~15KB | Definiciones de todas las tools |
| MCPs + Engram context | ~10KB | context7, firecrawl, engram, filesystem |
| **TOTAL** | **~130KB** | |

## El Culpable Principal

La lista `<available_skills>` con **205 skills** se genera automáticamente al escanear `~/.config/opencode/skills/`. opencode **no tiene lazy loading de metadatos** — si está en `skills/`, aparece en el system prompt. Punto.

## La Visión de Maestro

> Maestro debe gestionar todas las skills externamente. opencode solo tiene skills mínimos. Maestro resuelve qué cargar según contexto y proyecto.

### Flujo deseado

```
proyecto X → maestro scan → detecta stack (Go, React, etc.)
    → maestro resuelve skills necesarias
    → maestro las carga temporalmente en skills/ de opencode
    → opencode arranca limpio, solo con lo necesario
```

### Lo que maestro ya tiene

Paquetes en `internal/` relevantes:
- **`discover/`** — descubrimiento de capabilities
- **`classifier/`** — clasificación de skills por categoría
- **`routing/`** — ruteo inteligente
- **`db/`** — BD SQLite centralizada
- **`sources/`** — fuentes de skills (claude-skills, ECC, gstack, etc.)
- **`sync/`** — sync bidireccional
- **`heal/`** — auto-reparación
- **`compress/`** — compresión

## Lo Que Maestro Necesita Hacer

### Fase 1 — Skill Registry Manager (ahora)

Maestro necesita un subsistema `skill` que:

1. **Escanee todas las fuentes** de skills:
   - `~/tools/claude-skills/skills/`
   - `~/tools/ECC/` (ECC skills + RULES + WORKING-CONTEXT)
   - `~/tools/gstack/.opencode/skills/` (54 gstack skills)
   - `~/tools/taste-skill/skills/`
   - `~/tools/caveman/skills/`
   - `~/tools/gentle-ai/skills/`
   - `~/tools/opencode-power-pack/skills/`
   - `~/tools/gentleman-skills/`
   - `~/tools/skills/skills/` (community skills)
   - `~/tools/superpowers/`
   - `~/tools/last30days-skill/skills/`

2. **Mantenga un registro central** (SQLite) con:
   - Nombre, descripción, ruta real del SKILL.md
   - Categoría/source (framework, design, gstack, ecc, etc.)
   - Metadata: tamaño, dependencias, triggers
   - Frecuencia de uso
   - Tags de stack (react, go, python, docker, etc.)

3. **Comando `maestro skill`**:
   - `maestro skill list` — lista todas las skills conocidas
   - `maestro skill search <query>` — busca por nombre/descripción
   - `maestro skill install <name>` — crea symlink en `skills/` de opencode
   - `maestro skill remove <name>` — quita symlink de `skills/`
   - `maestro skill context-estimate` — calcula cuánto pesa el contexto actual
   - `maestro skill auto` — basado en el proyecto actual, instala skills relevantes

### Fase 2 — Context-Aware Resolver

Cuando maestro tenga daemon:

1. **MCP Server** que opencode consulta para resolver skills
2. **Router** que decide qué skills cargar según:
   - Stack del proyecto (detectado por `discover/`)
   - Historial de uso
   - Modelo actual (contexto limitado vs grande)
3. **Compresión automática** — skills de más de 50KB se cargan comprimidas

### Fase 3 — Ecosystem

1. Plugins para IDE/CLI
2. Marketplace de configs compartidas
3. Auto-tuning basado en uso real

## Stack Técnico

- **Lenguaje**: Go (ya en marcha)
- **Base de datos**: SQLite via modernc.org/sqlite (CGo-free, WAL mode)
- **CLI**: Cobra (ya implementado)
- **Daemon**: HTTP server con MCP protocol
- **Routing**: golang.org/x/sync para concurrencia

## Contexto Adicional

### Skills más pesados (por SKILL.md)

| Skill | Tamaño |
|---|---|
| gstack-ship | 171KB |
| gstack-plan-ceo-review | 143KB |
| last30days | 130KB |
| gstack-office-hours | 120KB |
| gstack-plan-devex-review | 118KB |
| gstack-plan-design-review | 116KB |
| gstack-spec | 116KB |
| gstack-plan-eng-review | 114KB |
| gstack-review | 100KB |
| gstack-design-review | 99KB |
| gstack-land-and-deploy | 95KB |
| gstack-autoplan | 95KB |
| taste-skill | 87KB |
| gstack-cso | 82KB |

### Skills por fuente

- **claude-skills**: ~40 skills (frameworks: react, vue, django, rails, rust, go, etc.)
- **gstack**: 54 skills (QA, design, deploy, review, etc.)
- **ECC**: ~50 skills (infra, backend, frontend, data, etc.)
- **taste-skill**: 13 skills (design, branding, image gen)
- **caveman**: 7 skills (compresión)
- **gentle-ai**: 7 skills (PRs, issues)
- **opencode-power-pack**: 8 skills (code review, feature dev)
- **gentleman-skills**: 12 skills (SDD cycle)
- **community**: ~20 skills (engineering, misc)
- **superpowers**: 14 skills (TDD, debugging, planning)
- **last30days**: 1 skill (130KB!)

### Comandos (104 total)

- ~30 gstack commands (qa, ship, plan-*, design-*, ios-*, etc.)
- ~40 ECC commands
- ~10 SDD commands
- ~10 gentle-ai commands
- ~10 misc commands
- ~4 builtin (ralph-loop, etc.)

## Próximo Paso Inmediato

1. Entrar a `~/projects/personal/maestro/`
2. Cargar skill: `skill maestro-dev`
3. Crear el paquete `internal/skill/` con:
   - `registry.go` — estructura de datos del registro de skills
   - `scanner.go` — escanea fuentes y construye registro
   - `manager.go` — instala/remueve symlinks en opencode
4. Comando Cobra: `cmd/maestro/skill.go`

## Referencias

- Memoria guardada en Engram con topic_key: `discovery/contexto-problema-de-skills-en-opencode-140kb-baseline`
- Archivo: `~/.config/opencode/skills/` (205 symlinks actuales)
- Config: `~/.config/opencode/opencode.json` (78 agents, references)
- Proyecto: `~/tools/ECC/` (WORKING-CONTEXT.md de 30KB)

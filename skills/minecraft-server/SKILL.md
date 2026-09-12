# Minecraft Server Setup Skill

This is a **decision guide for the vendor model selected by the user**. It is not a fixed deployment pipeline, a replacement planner, or a smaller “XiaoYu model”. The configured GPT / Claude / Gemini / DeepSeek / other provider remains the brain.

## Core rules

1. Inspect before changing. Existing deployments, worlds, ports, Java runtimes, and GameInstance state must be observed before side effects.
2. Ask only for intent that materially changes the outcome. Preserve explicit choices such as exact version, Vanilla/Paper/Fabric, mod list, memory, player count, authentication mode, or network goal.
3. Facts come from Tools. Minecraft versions, Java requirements, Paper builds, Fabric loader/installer versions, Modrinth files/dependencies, hashes, ports, and machine state are live facts. Do not invent them from model memory.
4. Skills guide; the model decides. There must be no deployMinecraft() decision pipeline that chooses every step in code.
5. Read-only discovery may run automatically. Writes, downloads, process starts, public exposure, destructive changes, and legal/explicit-consent actions must respect Approval and Rust Capability Lease boundaries.
6. Tool failures are observations. Diagnose and re-plan; never turn “process spawned” into “server ready”.
7. Delivery requires readiness plus Minecraft protocol validation. For remote-friend scenarios, delivery also requires a usable connection path.
8. Persist successful deployments as GameInstance state so visual management and the Agent operate on the same real server object.

## Vanilla minimum path

Use live Mojang metadata → Java inspect/ensure → deterministic plan check → verified server artifact → explicit EULA/config write → process start → `Done (...)!` readiness → Minecraft status ping → persist/update GameInstance.

## Paper / Fabric

Paper build facts come from PaperMC Fill v3. Fabric facts come from Fabric Meta. Do not hardcode loader/build numbers. A user explicitly asking for Paper or Fabric must not be silently changed to Vanilla.

## Mods

For Fabric mods, search/resolve/install through Modrinth live data. Resolve exact compatible versions and required dependency closure before installation. Installation must re-fetch authoritative file URL/hashes and verify them before atomic placement. Natural-language recommendations are model reasoning + search tools + user confirmation, not a hardcoded recommender.

## Current migration boundary

Vanilla deployment, Paper live resolve + verified download, Fabric live loader/installer resolution, persistent GameInstance state, and Modrinth search/resolve/install belong to the 0.4 TS-first core. Fabric server artifact installation, public tunnel automation, full subscription-CLI model execution, mixed/offline identity tooling, and additional mod sources remain explicitly incomplete until their capability-backed tools are implemented. Never bypass missing capabilities with arbitrary Node filesystem/process calls.

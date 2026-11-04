# Hardware sensors

Sensors is a read-only macOS view. It never changes fan targets or controls hardware. Energy retains battery status, power and sleep assertions.

## Readings and history

- CPU, GPU, SSD/NAND and battery summaries show the maximum valid mapped channel in each category. The same values are stored by the Go monitor at the existing sampling interval, within the existing history retention. Range and pause controls behave like other metric pages; background sampling continues while the window is hidden.
- CPU/GPU groups expand into individual raw channel IDs. They are not asserted to be individual physical cores. Battery, NAND, left/right airflow and wireless channels appear when exposed by hardware. The HID PMU die maximum is separate from CPU/GPU readings.
- Fans expose current, target, reported minimum and maximum RPM. Zero RPM is a valid stopped-fan reading. Each unavailable field remains unavailable independently; absence of a reading does not prove a fanless device.
- The page displays temperature and fan readings only. Independent power, voltage and current channels are still collected in Go but are not shown as an advanced electrical panel. Energy retains its existing power telemetry.
- Current temperature values and their trend lines share a color based on the displayed reading: green below 60°C, orange from 60°C, red from 80°C; missing readings are gray. These visual bands are configured in root `rockshinops.json` under `temperatureColorThresholds` and do not represent model-specific alarm limits, health or throttling. The numeric legend explains the ranges; macOS thermal pressure remains independent.
- Source failures and invalid temperatures produce null values, never zero or the previous reading. A discovered temperature channel with an invalid sample remains visible as unavailable. Charts break across null samples and sampling gaps rather than interpolating them.

## Mapping and limits

Mappings use the open-source [Stats sensor catalog](https://github.com/exelban/stats/blob/master/Modules/Sensors/values.swift), inspected on 2026-11-04. This is reverse-engineered component mapping, not an Apple API guarantee. CPU/GPU keys are gated by Intel or exact Apple M1/M2/M3/M4 chip generation, including the M4 versus M4 Pro/Max GPU distinction. Unknown CPU generations are not guessed. Common battery, NAND and airflow keys are probed independently. Generic Intel disk temperature channels are labeled storage; they are not assumed to be SSD/NAND.

The collector retains the existing conservative 10–125 °C validity range for temperature channels. Out-of-range and non-finite measurements are unavailable. This is an input-validity rule, not a health threshold; plausible firmware placeholders cannot be conclusively distinguished from real readings. No temperature health score is inferred. System thermal pressure is the separate macOS report.

M4 Pro hardware was probed locally. Other supported mappings require real-device acceptance. Windows/Linux sensor collection remains explicitly unsupported; other platform metrics are unchanged.

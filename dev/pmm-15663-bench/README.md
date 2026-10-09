# PMM-15663 annotation benchmark (scratch, not for merge)

Compares scoped annotations stored in Grafana (A, today), as `pmm_annotation_event` series in
VictoriaMetrics (E) and as rows in a ClickHouse `pmm.annotations` table (K).

- `gen.py` synthetic fleet (1000 nodes, 8000 services, 5 environments; `--stress` adds ~2.8M series) and annotations
- `load.sh` / `ann_load.py` load metrics and annotations into a PMM server (run on the VM host)
- `variants.py` / `upload.py` build and upload main / PR / E / K dashboard variants
- `capture.js` loads a dashboard in Chromium and records requests and annotation results
- `check.py` / `matrix.py` correctness matrix against the expected annotation set
- `bench.py` annotation query cost per store, median of 5, caches off

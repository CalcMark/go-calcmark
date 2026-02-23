---
title: "Network & Storage"
summary: "Latency, throughput, transfer time, read/seek, and compression functions."
weight: 20
---

From [`testdata/eval/success/features/network_functions.cm`](https://github.com/CalcMark/go-calcmark/blob/main/testdata/eval/success/features/network_functions.cm),
[`testdata/eval/success/features/storage_functions.cm`](https://github.com/CalcMark/go-calcmark/blob/main/testdata/eval/success/features/storage_functions.cm),
and [`testdata/eval/success/features/compression.cm`](https://github.com/CalcMark/go-calcmark/blob/main/testdata/eval/success/features/compression.cm).

## Network Latency (RTT)

```cm
local_latency = rtt(local)
regional_latency = rtt(regional)
continental_latency = rtt(continental)
global_latency = rtt(global)
```

## Network Throughput

```cm
gigabit_speed = throughput(gigabit)
ten_gig_speed = throughput(ten_gig)
hundred_gig = throughput(hundred_gig)
wifi_speed = throughput(wifi)
four_g_speed = throughput(four_g)
five_g_speed = throughput(five_g)
```

## Transfer Time (RTT + Transmission)

```cm
api_call = transfer_time(1 KB, regional, gigabit)
file_download = transfer_time(1 GB, global, gigabit)
video_chunk = transfer_time(10 MB, regional, ten_gig)
large_file = transfer_time(500 MB, continental, gigabit)

# Use in calculations
total_latency = rtt(regional) + 5 ms
throughput_check = throughput(gigabit) * 0.9
```

## Storage Read Times

```cm
ssd_read_100mb = read(100 MB, ssd)
nvme_read_1gb = read(1 GB, nvme)
hdd_read_10mb = read(10 MB, hdd)
pcie_read_500gb = read(500 GB, pcie_ssd)
sata_read = read(50 MB, sata_ssd)
```

## Seek/Access Latency

```cm
hdd_seek = seek(hdd)
ssd_seek = seek(ssd)
nvme_seek = seek(nvme)
pcie_seek = seek(pcie_ssd)
sata_seek = seek(sata_ssd)
```

## Combined Storage Operations

```cm
db_query_hdd = seek(hdd) + read(5 MB, hdd)
cache_hit_ssd = seek(ssd) + read(1 MB, ssd)
sequential_scan = read(100 GB, nvme)
total_io_time = seek(hdd) * 100 + read(5 GB, hdd)
```

## Compression Estimates

```cm
gzip_compressed = compress(1 GB, gzip)
lz4_compressed = compress(100 MB, lz4)
zstd_compressed = compress(500 MB, zstd)
bzip2_compressed = compress(1000 MB, bzip2)
snappy_compressed = compress(300 MB, snappy)
no_compression = compress(200 MB, none)

# Use in calculations
storage_savings = 10 GB - compress(10 GB, gzip)
compressed_transfer = transfer_time(compress(1 GB, lz4), global, gigabit)
```

### What This Demonstrates

- All network scope constants: `local`, `regional`, `continental`, `global`
- All connection types: `gigabit`, `ten_gig`, `hundred_gig`, `wifi`, `four_g`, `five_g`
- All storage types: `ssd`, `nvme`, `hdd`, `pcie_ssd`, `sata_ssd`
- All compression algorithms: `gzip`, `lz4`, `zstd`, `bzip2`, `snappy`, `none`
- Composing functions in arithmetic expressions
- Passing function results into other functions

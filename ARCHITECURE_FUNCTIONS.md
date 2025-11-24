# CalcMark Language Extensions for System Architecture

Based on Jeff Dean's "Numbers Everyone Should Know" (originally presented at Google, circa 2010, with updates for modern hardware), these extensions provide system architects with intuitive functions for back-of-the-envelope calculations.

## 1. Rate Accumulation: `accumulate()`

**Function:** `accumulate(rate, time_period)`  
**Natural:** `rate over time_period`

Handles multiplication of rates by time periods with automatic unit conversion.

```cm
# Function syntax
daily_data = accumulate(100 MB/s, 1 day)        # → 8.64 TB
monthly_cost = accumulate($0.10/hour, 30 days)  # → $72
yearly_logs = accumulate(5 GB/day, 1 year)      # → 1.825 TB

# Natural syntax  
daily_data = 100 MB/s over 1 day                # → 8.64 TB
monthly_cost = $0.10/hour over 30 days          # → $72
yearly_logs = 5 GB/day over 1 year              # → 1.825 TB
```

### Behind the scenes
```
100 MB/s over 1 day:
- Converts 1 day → 86,400 seconds
- Multiplies: 100 MB × 86,400 = 8,640,000 MB
- Converts to appropriate unit: 8.64 TB
```

## 2. Rate Conversion: `convert_rate()`

**Function:** `convert_rate(amount, target_time_unit)`  
**Natural:** `amount per time_unit`

Converts between different time-based rates.

```cm
# Function syntax
qps = convert_rate(5 million/day, second)       # → 57.87/second
bandwidth = convert_rate(10 TB/month, second)   # → 3.86 MB/second
hourly = convert_rate(1000 req/s, hour)         # → 3.6M/hour

# Natural syntax
qps = 5 million/day per second                  # → 57.87/second
bandwidth = 10 TB/month per second              # → 3.86 MB/second
hourly = 1000 req/s per hour                    # → 3.6M/hour
```

### Behind the scenes
```
5 million/day per second:
- Identifies conversion: day → second
- Divides by 86,400 seconds/day
- Result: 5,000,000 ÷ 86,400 = 57.87/second
```

## 3. Capacity Planning: `requires()`

**Function:** `requires(load, capacity_per_unit, buffer=0%)`  
**Natural:** `load with unit_capacity [and N% buffer]`

Performs ceiling division with optional buffer.

```cm
# Function syntax
servers = requires(10000 req/s, 450 req/s)              # → 23 servers
servers = requires(10000 req/s, 450 req/s, buffer=20%)  # → 28 servers
disks = requires(10 TB, 2 TB, buffer=10%)               # → 6 disks

# Natural syntax
servers = 10000 req/s with 450 req/s capacity           # → 23 servers
servers = 10000 req/s with 450 req/s capacity and 20% buffer # → 28 servers
disks = 10 TB with 2 TB capacity and 10% buffer         # → 6 disks
```

### Behind the scenes
```
10000 req/s with 450 req/s capacity and 20% buffer:
- Apply buffer to load: 10000 × 1.2 = 12000
- Divide: 12000 ÷ 450 = 26.67...
- Apply ceiling: ⌈26.67⌉ = 27
- Result: 27 servers
```

## 4. Availability/Downtime: `downtime()`

**Function:** `downtime(availability_percent, time_period)`  
**Natural:** `availability_percent downtime per time_period`

Converts availability percentage to actual downtime.

```cm
# Function syntax
monthly = downtime(99.9%, month)        # → 43.2 minutes
yearly = downtime(99.99%, year)         # → 52.56 minutes  
daily = downtime(99.999%, day)          # → 0.864 seconds

# Natural syntax
monthly = 99.9% downtime per month      # → 43.2 minutes
yearly = 99.99% downtime per year       # → 52.56 minutes
daily = 99.999% downtime per day        # → 0.864 seconds
```

### Behind the scenes
```
99.9% downtime per month:
- Calculates unavailability: 100% - 99.9% = 0.1%
- Converts month to minutes: 30 days × 24 hours × 60 min = 43,200 minutes
- Multiplies: 43,200 × 0.001 = 43.2 minutes
```

## 5. Network Latency: `rtt()`

**Function:** `rtt(scope)`  
**Scopes:** `local`, `regional`, `continental`, `global`

Returns network round-trip time. This is constant regardless of data size.

```cm
# Function syntax
api_latency = rtt(regional) + 5 ms              # → 10 ms + 5 ms = 15 ms
db_latency = rtt(local) + seek(ssd)             # → 0.5 ms + 0.15 ms = 0.65 ms
user_latency = rtt(global) + processing         # → 150 ms + processing

# Natural syntax (if implemented)
api_latency = regional rtt + 5 ms               # → 15 ms
```

### Behind the scenes
```
rtt(global):
- Distance: ~10,000 km
- Speed of light in fiber: ~200,000 km/s
- Theoretical minimum: 10,000 km × 2 (round trip) ÷ 200,000 = 100 ms
- Add routing/switching overhead: ~50 ms
- Result: 150 ms
```

## 6. Storage Operations: `read()` and `seek()`

**Functions:** `read(type, size=1MB)`, `seek(type)`  
**Types:** `memory`, `ssd`, `disk`, `network`

Based on Jeff Dean's latency numbers, updated for modern hardware.

```cm
# Function syntax
cache_read = read(memory, 100KB)                # → 25 μs
db_read = read(ssd, 10MB)                       # → 10 ms
backup_read = read(disk, 100MB)                 # → 2 seconds
disk_latency = seek(disk)                       # → 10 ms
ssd_latency = seek(ssd)                         # → 150 μs

# Natural syntax (if implemented)
cache_read = read 100KB from memory             # → 25 μs
disk_latency = seek disk                        # → 10 ms

# Combined example
db_query = seek(disk) + read(disk, 5MB)         # → 10 ms + 100 ms = 110 ms
cache_hit = seek(memory) + read(memory, 1KB)    # → ~100 ns
```

### Behind the scenes
```
read(disk, 5MB):
- Disk sequential read: ~100 MB/s
- Time = 5 MB ÷ 100 MB/s = 50 ms
- Conservative estimate: 100 ms

seek(disk):
- Rotational latency: ~4 ms (7200 RPM)
- Seek time: ~6 ms
- Total: 10 ms

read(memory, 1MB):
- Memory bandwidth: ~4 GB/s
- Time = 1 MB ÷ 4 GB/s = 250 μs
```

## 7. Compression: `compress()`

**Function:** `compress(type, size)`  
**Types:** `text`, `json`, `image`, `video`, `binary`

Returns compressed size based on typical ratios.

```cm
# Function syntax
payload = compress(json, 100MB)                 # → 20 MB
images = compress(image, 50MB)                  # → 35 MB
logs = compress(text, 1GB)                      # → 100 MB

# Natural syntax (if implemented)
payload = 100MB json compressed                 # → 20 MB
```

### Behind the scenes
```
compress(json, 100MB):
- JSON typical compression ratio: 5:1
- Compressed size: 100 MB ÷ 5 = 20 MB
```

## 8. Network Throughput: `throughput()`

**Function:** `throughput(type)`  
**Types:** `gigabit`, `10g`, `100g`, `wifi`, `4g`, `5g`

```cm
# Function syntax
server_bw = throughput(10g)                     # → 1.25 GB/s
home_bw = throughput(gigabit)                   # → 125 MB/s
mobile_bw = throughput(5g)                      # → 50 MB/s

# Usage in calculations
servers = bandwidth with throughput(10g) capacity  # How many 10G ports needed
transfer_time = file_size / throughput(gigabit)    # Time to transfer
```

## 9. Data Transfer Time: `transfer_time()`

**Function:** `transfer_time(size, distance, connection)`

Combines network latency (RTT) with transmission time.

```cm
# Function syntax
api_call = transfer_time(1 KB, regional, gigabit)       # → 10 ms
file_download = transfer_time(1 GB, global, gigabit)    # → 8.15 seconds
video_chunk = transfer_time(10 MB, regional, 10g)       # → 18 ms

# Natural syntax (if implemented)
api_call = transfer 1 KB regional over gigabit          # → 10 ms
```

### Behind the scenes
```
transfer_time(1 GB, global, gigabit):
- Network RTT: rtt(global) = 150 ms
- Transmission: 1 GB ÷ 125 MB/s = 8 seconds
- Total: 150 ms + 8000 ms = 8150 ms
```

## 10. Napkin Rounding: `as napkin`

Rounds to human-friendly numbers using engineering notation. Uses the 1-2-5 sequence, a standard engineering practice that rounds to the pattern: 1, 2, 5, 10, 20, 50, 100, 200, 500, 1000, etc. This sequence ensures numbers are never more than 2x away from actual values and remain easy for mental math.

```cm
# Unit conversion syntax (like existing CalcMark)
servers = 47 as napkin                          # → 50
requests = 347234 as napkin                     # → 350K  
bandwidth = 8734 MB/s as napkin                 # → 9 GB/s (rounds up)
storage = 2.347 PB as napkin                    # → 2.5 PB (1-2-5 sequence)
latency = 161.65 ms as napkin                   # → 160 ms (rounds to nearest)
cost = $4,789,234 as napkin                     # → $5M
```

### Behind the scenes
```
347234 as napkin using 1-2-5 sequence:
- Find order of magnitude: 10^5 (hundreds of thousands)
- Nearest 1-2-5 values: 200K, 350K, 500K
- Closest match: 350K
- This is easier to work with than 347K or 340K

Why 1-2-5? It matches how engineers think:
- 47 servers → "about 50"
- 2.3 PB → "two and a half petabytes"  
- 8.7 Gbps → "roughly 9 gigs" or "almost 10 gigs"
```

## Complete Example: Video Streaming Platform

```cm
# Traffic estimation
users = 10 million
concurrent = users * 0.1                        # → 1M viewers
bitrate = 5 Mbps

# Function syntax for bandwidth
total_bw = accumulate(concurrent * bitrate, 1 second)    # → 5 Tbps
cdn_bw = total_bw * 0.9                                 # → 4.5 Tbps (90% from CDN)
origin_bw = total_bw * 0.1                              # → 500 Gbps (10% from origin)

# Natural syntax for bandwidth  
total_bw = concurrent * bitrate over 1 second           # → 5 Tbps
monthly_transfer = total_bw over 30 days                # → 1.62 EB

# Server capacity - function syntax
cdn_servers = requires(cdn_bw, throughput(10g), buffer=30%)     # → 585 servers
origin_servers = requires(origin_bw, throughput(10g), buffer=50%) # → 75 servers

# Server capacity - natural syntax
cdn_servers = cdn_bw with throughput(10g) capacity and 30% buffer    # → 585
origin_servers = origin_bw with throughput(10g) capacity and 50% buffer # → 75

# User experience - function syntax
startup_time = rtt(regional) + 
               transfer_time(2 MB, regional, gigabit)    # → 36 ms
cache_miss = rtt(regional) + rtt(local) + 
             seek(disk) + 
             transfer_time(2 MB, local, 10g)            # → 22.6 ms

# User experience - natural syntax (if implemented)
startup_time = regional rtt + 
               transfer 2 MB regional over gigabit       # → 36 ms

# Apply napkin rounding for presentation
cdn_servers as napkin                                   # → 600 servers (1-2-5: 585→600)
origin_servers as napkin                                # → 75 servers (already clean)
startup_time as napkin                                  # → 35 ms (36→35)
cache_miss as napkin                                    # → 25 ms (23→25)

# Availability - function syntax
sla = 99.95%
monthly_down = downtime(sla, month)                     # → 21.6 minutes
yearly_down = downtime(sla, year)                       # → 4.38 hours

# Availability - natural syntax
monthly_down = sla downtime per month                   # → 21.6 minutes
yearly_down = sla downtime per year                     # → 4.38 hours

# Storage growth
daily_uploads = 10000 videos * compress(video, 500 MB)  # → 4.75 TB
yearly_storage = accumulate(daily_uploads, 1 year)      # → 1.73 PB
# Or natural syntax
yearly_storage = daily_uploads over 1 year              # → 1.73 PB
yearly_storage as napkin                                # → 2 PB (1-2-5: 1.73→2)

# Monthly costs - note how napkin makes discussion easier
bandwidth_cost = monthly_transfer * $0.02/GB as napkin  # → $30M 
storage_cost = yearly_storage * $20/TB/month as napkin  # → $35K→$40K
total_cost = (bandwidth_cost + storage_cost) as napkin # → $30M (storage negligible)
```

## Database-Backed API Example

```cm
# Load estimation
users = 50 million
daily_active = users * 0.2                      # → 10M DAU
api_calls_per_user = 100/day
total_calls = daily_active * api_calls_per_user # → 1B/day

# Convert to QPS - both syntaxes
qps = convert_rate(total_calls, second)         # → 11,574/s
qps = total_calls per second                    # → 11,574/s
peak_qps = qps * 3                              # → 34,722/s
peak_qps as napkin                              # → 35K/s (1-2-5: 34.7K→35K)

# Latency breakdown using Jeff Dean's numbers
request_path = rtt(regional) +                  # User to datacenter: 10 ms
               rtt(local) * 2 +                  # Microservices: 1 ms
               seek(ssd) +                       # Database seek: 150 μs
               read(ssd, 10KB) +                 # Read data: 100 μs
               5 ms                              # Processing: 5 ms
# Total: 10 + 1 + 0.15 + 0.1 + 5 = 16.25 ms
request_path as napkin                          # → 16 ms (rounds to nearest)

# Capacity planning - both syntaxes
server_capacity = 1000 req/s
servers = requires(peak_qps, server_capacity, buffer=30%)  # → 46 servers
servers = peak_qps with server_capacity and 30% buffer     # → 46 servers
servers as napkin                                          # → 50 servers (1-2-5: 46→50)

# Network bandwidth
response_size = 5 KB
bandwidth_needed = peak_qps * response_size                # → 174 MB/s
bandwidth_needed as napkin                                 # → 200 MB/s (1-2-5: 174→200)
network_cards = bandwidth_needed with throughput(gigabit) capacity  # → 2
```

These extensions, grounded in Jeff Dean's performance numbers, make CalcMark a powerful tool for system architecture estimates while keeping the syntax intuitive and calculations transparent. The napkin rounding follows engineering conventions to produce numbers that are both accurate enough and easy to work with.

window.BENCHMARK_DATA = {
  "lastUpdate": 1749398598358,
  "repoUrl": "https://github.com/pilotso11/lazywritercache",
  "entries": {
    "Go Benchmark": [
      {
        "commit": {
          "author": {
            "name": "pilotso11",
            "username": "pilotso11"
          },
          "committer": {
            "name": "pilotso11",
            "username": "pilotso11"
          },
          "id": "582c1b031970a2937647b2bafb8ae9e1b17c0b3c",
          "message": "Update go.yml to add benchmarks",
          "timestamp": "2025-06-04T22:31:40Z",
          "url": "https://github.com/pilotso11/lazywritercache/pull/10/commits/582c1b031970a2937647b2bafb8ae9e1b17c0b3c"
        },
        "date": 1749376495643,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkCacheWriteMax20k",
            "value": 287.2,
            "unit": "ns/op\t     150 B/op\t       3 allocs/op",
            "extra": "4364035 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax20k - ns/op",
            "value": 287.2,
            "unit": "ns/op",
            "extra": "4364035 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax20k - B/op",
            "value": 150,
            "unit": "B/op",
            "extra": "4364035 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax20k - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "4364035 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax100k",
            "value": 284.7,
            "unit": "ns/op\t     141 B/op\t       3 allocs/op",
            "extra": "4028944 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax100k - ns/op",
            "value": 284.7,
            "unit": "ns/op",
            "extra": "4028944 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax100k - B/op",
            "value": 141,
            "unit": "B/op",
            "extra": "4028944 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax100k - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "4028944 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead20k",
            "value": 82.09,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "12982452 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead20k - ns/op",
            "value": 82.09,
            "unit": "ns/op",
            "extra": "12982452 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead20k - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "12982452 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead20k - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "12982452 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead100k",
            "value": 109.3,
            "unit": "ns/op\t       3 B/op\t       0 allocs/op",
            "extra": "10998994 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead100k - ns/op",
            "value": 109.3,
            "unit": "ns/op",
            "extra": "10998994 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead100k - B/op",
            "value": 3,
            "unit": "B/op",
            "extra": "10998994 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead100k - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "10998994 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x5_CacheRead20k",
            "value": 703,
            "unit": "ns/op\t       5 B/op\t       0 allocs/op",
            "extra": "1696614 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x5_CacheRead20k - ns/op",
            "value": 703,
            "unit": "ns/op",
            "extra": "1696614 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x5_CacheRead20k - B/op",
            "value": 5,
            "unit": "B/op",
            "extra": "1696614 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x5_CacheRead20k - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "1696614 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x10_CacheRead20k",
            "value": 1659,
            "unit": "ns/op\t      12 B/op\t       0 allocs/op",
            "extra": "709537 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x10_CacheRead20k - ns/op",
            "value": 1659,
            "unit": "ns/op",
            "extra": "709537 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x10_CacheRead20k - B/op",
            "value": 12,
            "unit": "B/op",
            "extra": "709537 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x10_CacheRead20k - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "709537 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "pilotso11",
            "username": "pilotso11"
          },
          "committer": {
            "name": "pilotso11",
            "username": "pilotso11"
          },
          "id": "d745531862e3fe7a62bb9fb3f533e5c2d96a7588",
          "message": "Update go.yml to add benchmarks",
          "timestamp": "2025-06-04T22:31:40Z",
          "url": "https://github.com/pilotso11/lazywritercache/pull/10/commits/d745531862e3fe7a62bb9fb3f533e5c2d96a7588"
        },
        "date": 1749376762401,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkCacheWriteMax20k",
            "value": 272.1,
            "unit": "ns/op\t     149 B/op\t       3 allocs/op",
            "extra": "4392747 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax20k - ns/op",
            "value": 272.1,
            "unit": "ns/op",
            "extra": "4392747 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax20k - B/op",
            "value": 149,
            "unit": "B/op",
            "extra": "4392747 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax20k - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "4392747 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax100k",
            "value": 280,
            "unit": "ns/op\t     142 B/op\t       3 allocs/op",
            "extra": "3959464 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax100k - ns/op",
            "value": 280,
            "unit": "ns/op",
            "extra": "3959464 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax100k - B/op",
            "value": 142,
            "unit": "B/op",
            "extra": "3959464 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax100k - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "3959464 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead20k",
            "value": 81.5,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "13112714 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead20k - ns/op",
            "value": 81.5,
            "unit": "ns/op",
            "extra": "13112714 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead20k - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "13112714 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead20k - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "13112714 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead100k",
            "value": 108.6,
            "unit": "ns/op\t       3 B/op\t       0 allocs/op",
            "extra": "10396686 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead100k - ns/op",
            "value": 108.6,
            "unit": "ns/op",
            "extra": "10396686 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead100k - B/op",
            "value": 3,
            "unit": "B/op",
            "extra": "10396686 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead100k - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "10396686 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x5_CacheRead20k",
            "value": 691,
            "unit": "ns/op\t       5 B/op\t       0 allocs/op",
            "extra": "1708071 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x5_CacheRead20k - ns/op",
            "value": 691,
            "unit": "ns/op",
            "extra": "1708071 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x5_CacheRead20k - B/op",
            "value": 5,
            "unit": "B/op",
            "extra": "1708071 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x5_CacheRead20k - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "1708071 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x10_CacheRead20k",
            "value": 1661,
            "unit": "ns/op\t      11 B/op\t       0 allocs/op",
            "extra": "728703 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x10_CacheRead20k - ns/op",
            "value": 1661,
            "unit": "ns/op",
            "extra": "728703 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x10_CacheRead20k - B/op",
            "value": 11,
            "unit": "B/op",
            "extra": "728703 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x10_CacheRead20k - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "728703 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "pilotso11",
            "username": "pilotso11"
          },
          "committer": {
            "name": "pilotso11",
            "username": "pilotso11"
          },
          "id": "157557646f0a0fb88bdd2a7e560737846ee1a1d4",
          "message": "Update go.yml to add benchmarks",
          "timestamp": "2025-06-04T22:31:40Z",
          "url": "https://github.com/pilotso11/lazywritercache/pull/10/commits/157557646f0a0fb88bdd2a7e560737846ee1a1d4"
        },
        "date": 1749376815176,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkCacheWriteMax20k",
            "value": 266.2,
            "unit": "ns/op\t     135 B/op\t       3 allocs/op",
            "extra": "4126905 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax20k - ns/op",
            "value": 266.2,
            "unit": "ns/op",
            "extra": "4126905 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax20k - B/op",
            "value": 135,
            "unit": "B/op",
            "extra": "4126905 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax20k - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "4126905 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax100k",
            "value": 308.8,
            "unit": "ns/op\t     141 B/op\t       3 allocs/op",
            "extra": "3267861 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax100k - ns/op",
            "value": 308.8,
            "unit": "ns/op",
            "extra": "3267861 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax100k - B/op",
            "value": 141,
            "unit": "B/op",
            "extra": "3267861 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax100k - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "3267861 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead20k",
            "value": 82.02,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "13156359 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead20k - ns/op",
            "value": 82.02,
            "unit": "ns/op",
            "extra": "13156359 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead20k - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "13156359 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead20k - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "13156359 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead100k",
            "value": 108.1,
            "unit": "ns/op\t       4 B/op\t       0 allocs/op",
            "extra": "9565578 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead100k - ns/op",
            "value": 108.1,
            "unit": "ns/op",
            "extra": "9565578 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead100k - B/op",
            "value": 4,
            "unit": "B/op",
            "extra": "9565578 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead100k - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "9565578 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x5_CacheRead20k",
            "value": 706,
            "unit": "ns/op\t       5 B/op\t       0 allocs/op",
            "extra": "1707498 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x5_CacheRead20k - ns/op",
            "value": 706,
            "unit": "ns/op",
            "extra": "1707498 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x5_CacheRead20k - B/op",
            "value": 5,
            "unit": "B/op",
            "extra": "1707498 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x5_CacheRead20k - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "1707498 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x10_CacheRead20k",
            "value": 1702,
            "unit": "ns/op\t      11 B/op\t       0 allocs/op",
            "extra": "718510 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x10_CacheRead20k - ns/op",
            "value": 1702,
            "unit": "ns/op",
            "extra": "718510 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x10_CacheRead20k - B/op",
            "value": 11,
            "unit": "B/op",
            "extra": "718510 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x10_CacheRead20k - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "718510 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "pilotso11",
            "username": "pilotso11"
          },
          "committer": {
            "name": "pilotso11",
            "username": "pilotso11"
          },
          "id": "12233f0dbcc4969f5c79f532d89b8ee4b24a251c",
          "message": "Update go.yml to add benchmarks",
          "timestamp": "2025-06-04T22:31:40Z",
          "url": "https://github.com/pilotso11/lazywritercache/pull/10/commits/12233f0dbcc4969f5c79f532d89b8ee4b24a251c"
        },
        "date": 1749376873755,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkCacheWriteMax20k",
            "value": 272.3,
            "unit": "ns/op\t     149 B/op\t       3 allocs/op",
            "extra": "4411953 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax20k - ns/op",
            "value": 272.3,
            "unit": "ns/op",
            "extra": "4411953 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax20k - B/op",
            "value": 149,
            "unit": "B/op",
            "extra": "4411953 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax20k - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "4411953 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax100k",
            "value": 275.6,
            "unit": "ns/op\t     140 B/op\t       3 allocs/op",
            "extra": "4054426 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax100k - ns/op",
            "value": 275.6,
            "unit": "ns/op",
            "extra": "4054426 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax100k - B/op",
            "value": 140,
            "unit": "B/op",
            "extra": "4054426 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax100k - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "4054426 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead20k",
            "value": 81.13,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "13083759 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead20k - ns/op",
            "value": 81.13,
            "unit": "ns/op",
            "extra": "13083759 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead20k - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "13083759 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead20k - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "13083759 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead100k",
            "value": 108.3,
            "unit": "ns/op\t       3 B/op\t       0 allocs/op",
            "extra": "10381230 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead100k - ns/op",
            "value": 108.3,
            "unit": "ns/op",
            "extra": "10381230 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead100k - B/op",
            "value": 3,
            "unit": "B/op",
            "extra": "10381230 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead100k - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "10381230 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x5_CacheRead20k",
            "value": 699.3,
            "unit": "ns/op\t       5 B/op\t       0 allocs/op",
            "extra": "1696209 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x5_CacheRead20k - ns/op",
            "value": 699.3,
            "unit": "ns/op",
            "extra": "1696209 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x5_CacheRead20k - B/op",
            "value": 5,
            "unit": "B/op",
            "extra": "1696209 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x5_CacheRead20k - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "1696209 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x10_CacheRead20k",
            "value": 1681,
            "unit": "ns/op\t      12 B/op\t       0 allocs/op",
            "extra": "708169 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x10_CacheRead20k - ns/op",
            "value": 1681,
            "unit": "ns/op",
            "extra": "708169 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x10_CacheRead20k - B/op",
            "value": 12,
            "unit": "B/op",
            "extra": "708169 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x10_CacheRead20k - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "708169 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "email": "seth@oshers.com",
            "name": "pilotso11",
            "username": "pilotso11"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "56c16a8abb2e758dae762c84b04f99bab8115992",
          "message": "Merge pull request #10 from pilotso11/pilotso11-patch-1\n\nUpdate go.yml to add benchmarks",
          "timestamp": "2025-06-08T11:02:01+01:00",
          "tree_id": "254bc8a9fb35223ffbb600b82139cb477584bb8a",
          "url": "https://github.com/pilotso11/lazywritercache/commit/56c16a8abb2e758dae762c84b04f99bab8115992"
        },
        "date": 1749376964500,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkCacheWriteMax20k",
            "value": 279.1,
            "unit": "ns/op\t     149 B/op\t       3 allocs/op",
            "extra": "4426376 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax20k - ns/op",
            "value": 279.1,
            "unit": "ns/op",
            "extra": "4426376 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax20k - B/op",
            "value": 149,
            "unit": "B/op",
            "extra": "4426376 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax20k - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "4426376 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax100k",
            "value": 281.6,
            "unit": "ns/op\t     141 B/op\t       3 allocs/op",
            "extra": "4048660 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax100k - ns/op",
            "value": 281.6,
            "unit": "ns/op",
            "extra": "4048660 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax100k - B/op",
            "value": 141,
            "unit": "B/op",
            "extra": "4048660 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax100k - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "4048660 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead20k",
            "value": 81,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "13140746 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead20k - ns/op",
            "value": 81,
            "unit": "ns/op",
            "extra": "13140746 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead20k - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "13140746 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead20k - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "13140746 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead100k",
            "value": 108,
            "unit": "ns/op\t       4 B/op\t       0 allocs/op",
            "extra": "10115096 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead100k - ns/op",
            "value": 108,
            "unit": "ns/op",
            "extra": "10115096 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead100k - B/op",
            "value": 4,
            "unit": "B/op",
            "extra": "10115096 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead100k - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "10115096 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x5_CacheRead20k",
            "value": 698.6,
            "unit": "ns/op\t       5 B/op\t       0 allocs/op",
            "extra": "1695924 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x5_CacheRead20k - ns/op",
            "value": 698.6,
            "unit": "ns/op",
            "extra": "1695924 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x5_CacheRead20k - B/op",
            "value": 5,
            "unit": "B/op",
            "extra": "1695924 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x5_CacheRead20k - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "1695924 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x10_CacheRead20k",
            "value": 1659,
            "unit": "ns/op\t      11 B/op\t       0 allocs/op",
            "extra": "717262 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x10_CacheRead20k - ns/op",
            "value": 1659,
            "unit": "ns/op",
            "extra": "717262 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x10_CacheRead20k - B/op",
            "value": 11,
            "unit": "B/op",
            "extra": "717262 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x10_CacheRead20k - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "717262 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "pilotso11",
            "username": "pilotso11"
          },
          "committer": {
            "name": "pilotso11",
            "username": "pilotso11"
          },
          "id": "1b97835d21f45e92e7650ff294ed879303674361",
          "message": "Improve locking",
          "timestamp": "2025-06-08T10:02:05Z",
          "url": "https://github.com/pilotso11/lazywritercache/pull/9/commits/1b97835d21f45e92e7650ff294ed879303674361"
        },
        "date": 1749377434111,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkCacheWriteMax20kLF",
            "value": 327.1,
            "unit": "ns/op\t     109 B/op\t       6 allocs/op",
            "extra": "3574633 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax20kLF - ns/op",
            "value": 327.1,
            "unit": "ns/op",
            "extra": "3574633 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax20kLF - B/op",
            "value": 109,
            "unit": "B/op",
            "extra": "3574633 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax20kLF - allocs/op",
            "value": 6,
            "unit": "allocs/op",
            "extra": "3574633 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax100kLF",
            "value": 394.7,
            "unit": "ns/op\t     115 B/op\t       7 allocs/op",
            "extra": "2958865 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax100kLF - ns/op",
            "value": 394.7,
            "unit": "ns/op",
            "extra": "2958865 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax100kLF - B/op",
            "value": 115,
            "unit": "B/op",
            "extra": "2958865 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax100kLF - allocs/op",
            "value": 7,
            "unit": "allocs/op",
            "extra": "2958865 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead20kLF",
            "value": 78.6,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "16075204 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead20kLF - ns/op",
            "value": 78.6,
            "unit": "ns/op",
            "extra": "16075204 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead20kLF - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "16075204 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead20kLF - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "16075204 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead100kLF",
            "value": 126.9,
            "unit": "ns/op\t       4 B/op\t       0 allocs/op",
            "extra": "9074018 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead100kLF - ns/op",
            "value": 126.9,
            "unit": "ns/op",
            "extra": "9074018 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead100kLF - B/op",
            "value": 4,
            "unit": "B/op",
            "extra": "9074018 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead100kLF - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "9074018 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x5_CacheRead20kLF",
            "value": 160.3,
            "unit": "ns/op\t       1 B/op\t       0 allocs/op",
            "extra": "6462334 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x5_CacheRead20kLF - ns/op",
            "value": 160.3,
            "unit": "ns/op",
            "extra": "6462334 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x5_CacheRead20kLF - B/op",
            "value": 1,
            "unit": "B/op",
            "extra": "6462334 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x5_CacheRead20kLF - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "6462334 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x10_CacheRead20kLF",
            "value": 321.9,
            "unit": "ns/op\t       2 B/op\t       0 allocs/op",
            "extra": "3332476 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x10_CacheRead20kLF - ns/op",
            "value": 321.9,
            "unit": "ns/op",
            "extra": "3332476 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x10_CacheRead20kLF - B/op",
            "value": 2,
            "unit": "B/op",
            "extra": "3332476 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x10_CacheRead20kLF - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "3332476 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x20_CacheRead20kLF",
            "value": 639.7,
            "unit": "ns/op\t       4 B/op\t       0 allocs/op",
            "extra": "1808064 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x20_CacheRead20kLF - ns/op",
            "value": 639.7,
            "unit": "ns/op",
            "extra": "1808064 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x20_CacheRead20kLF - B/op",
            "value": 4,
            "unit": "B/op",
            "extra": "1808064 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x20_CacheRead20kLF - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "1808064 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax20k",
            "value": 280.6,
            "unit": "ns/op\t     149 B/op\t       3 allocs/op",
            "extra": "4416446 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax20k - ns/op",
            "value": 280.6,
            "unit": "ns/op",
            "extra": "4416446 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax20k - B/op",
            "value": 149,
            "unit": "B/op",
            "extra": "4416446 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax20k - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "4416446 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax100k",
            "value": 289.3,
            "unit": "ns/op\t     142 B/op\t       3 allocs/op",
            "extra": "3959712 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax100k - ns/op",
            "value": 289.3,
            "unit": "ns/op",
            "extra": "3959712 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax100k - B/op",
            "value": 142,
            "unit": "B/op",
            "extra": "3959712 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax100k - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "3959712 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead20k",
            "value": 80.41,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "13348135 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead20k - ns/op",
            "value": 80.41,
            "unit": "ns/op",
            "extra": "13348135 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead20k - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "13348135 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead20k - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "13348135 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead100k",
            "value": 113.8,
            "unit": "ns/op\t       4 B/op\t       0 allocs/op",
            "extra": "10043428 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead100k - ns/op",
            "value": 113.8,
            "unit": "ns/op",
            "extra": "10043428 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead100k - B/op",
            "value": 4,
            "unit": "B/op",
            "extra": "10043428 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead100k - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "10043428 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x5_CacheRead20k",
            "value": 687.7,
            "unit": "ns/op\t       4 B/op\t       0 allocs/op",
            "extra": "1717802 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x5_CacheRead20k - ns/op",
            "value": 687.7,
            "unit": "ns/op",
            "extra": "1717802 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x5_CacheRead20k - B/op",
            "value": 4,
            "unit": "B/op",
            "extra": "1717802 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x5_CacheRead20k - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "1717802 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x10_CacheRead20k",
            "value": 1634,
            "unit": "ns/op\t      11 B/op\t       0 allocs/op",
            "extra": "727177 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x10_CacheRead20k - ns/op",
            "value": 1634,
            "unit": "ns/op",
            "extra": "727177 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x10_CacheRead20k - B/op",
            "value": 11,
            "unit": "B/op",
            "extra": "727177 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x10_CacheRead20k - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "727177 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x20_CacheRead20k",
            "value": 3984,
            "unit": "ns/op\t      34 B/op\t       0 allocs/op",
            "extra": "251257 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x20_CacheRead20k - ns/op",
            "value": 3984,
            "unit": "ns/op",
            "extra": "251257 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x20_CacheRead20k - B/op",
            "value": 34,
            "unit": "B/op",
            "extra": "251257 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x20_CacheRead20k - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "251257 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "email": "seth@oshers.com",
            "name": "pilotso11",
            "username": "pilotso11"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "06e257ff380d35b759ee5b53669e67728a13b459",
          "message": "Merge pull request #9 from pilotso11/improve-locking\n\nImprove locking",
          "timestamp": "2025-06-08T11:13:38+01:00",
          "tree_id": "192ad5212e67923a834c70ce91c4a37d6416be0a",
          "url": "https://github.com/pilotso11/lazywritercache/commit/06e257ff380d35b759ee5b53669e67728a13b459"
        },
        "date": 1749377680771,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkCacheWriteMax20kLF",
            "value": 327,
            "unit": "ns/op\t     109 B/op\t       6 allocs/op",
            "extra": "3538893 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax20kLF - ns/op",
            "value": 327,
            "unit": "ns/op",
            "extra": "3538893 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax20kLF - B/op",
            "value": 109,
            "unit": "B/op",
            "extra": "3538893 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax20kLF - allocs/op",
            "value": 6,
            "unit": "allocs/op",
            "extra": "3538893 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax100kLF",
            "value": 395.7,
            "unit": "ns/op\t     115 B/op\t       7 allocs/op",
            "extra": "2976208 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax100kLF - ns/op",
            "value": 395.7,
            "unit": "ns/op",
            "extra": "2976208 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax100kLF - B/op",
            "value": 115,
            "unit": "B/op",
            "extra": "2976208 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax100kLF - allocs/op",
            "value": 7,
            "unit": "allocs/op",
            "extra": "2976208 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead20kLF",
            "value": 73.98,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "16825731 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead20kLF - ns/op",
            "value": 73.98,
            "unit": "ns/op",
            "extra": "16825731 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead20kLF - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "16825731 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead20kLF - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "16825731 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead100kLF",
            "value": 101.4,
            "unit": "ns/op\t       3 B/op\t       0 allocs/op",
            "extra": "10580175 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead100kLF - ns/op",
            "value": 101.4,
            "unit": "ns/op",
            "extra": "10580175 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead100kLF - B/op",
            "value": 3,
            "unit": "B/op",
            "extra": "10580175 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead100kLF - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "10580175 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x5_CacheRead20kLF",
            "value": 161.7,
            "unit": "ns/op\t       1 B/op\t       0 allocs/op",
            "extra": "7302452 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x5_CacheRead20kLF - ns/op",
            "value": 161.7,
            "unit": "ns/op",
            "extra": "7302452 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x5_CacheRead20kLF - B/op",
            "value": 1,
            "unit": "B/op",
            "extra": "7302452 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x5_CacheRead20kLF - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "7302452 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x10_CacheRead20kLF",
            "value": 313.6,
            "unit": "ns/op\t       2 B/op\t       0 allocs/op",
            "extra": "3361472 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x10_CacheRead20kLF - ns/op",
            "value": 313.6,
            "unit": "ns/op",
            "extra": "3361472 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x10_CacheRead20kLF - B/op",
            "value": 2,
            "unit": "B/op",
            "extra": "3361472 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x10_CacheRead20kLF - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "3361472 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x20_CacheRead20kLF",
            "value": 627.7,
            "unit": "ns/op\t       4 B/op\t       0 allocs/op",
            "extra": "1835407 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x20_CacheRead20kLF - ns/op",
            "value": 627.7,
            "unit": "ns/op",
            "extra": "1835407 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x20_CacheRead20kLF - B/op",
            "value": 4,
            "unit": "B/op",
            "extra": "1835407 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x20_CacheRead20kLF - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "1835407 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax20k",
            "value": 267,
            "unit": "ns/op\t     151 B/op\t       3 allocs/op",
            "extra": "4338571 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax20k - ns/op",
            "value": 267,
            "unit": "ns/op",
            "extra": "4338571 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax20k - B/op",
            "value": 151,
            "unit": "B/op",
            "extra": "4338571 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax20k - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "4338571 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax100k",
            "value": 305.3,
            "unit": "ns/op\t     146 B/op\t       3 allocs/op",
            "extra": "3830572 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax100k - ns/op",
            "value": 305.3,
            "unit": "ns/op",
            "extra": "3830572 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax100k - B/op",
            "value": 146,
            "unit": "B/op",
            "extra": "3830572 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax100k - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "3830572 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead20k",
            "value": 80.09,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "13453686 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead20k - ns/op",
            "value": 80.09,
            "unit": "ns/op",
            "extra": "13453686 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead20k - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "13453686 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead20k - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "13453686 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead100k",
            "value": 109.1,
            "unit": "ns/op\t       4 B/op\t       0 allocs/op",
            "extra": "10097592 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead100k - ns/op",
            "value": 109.1,
            "unit": "ns/op",
            "extra": "10097592 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead100k - B/op",
            "value": 4,
            "unit": "B/op",
            "extra": "10097592 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead100k - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "10097592 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x5_CacheRead20k",
            "value": 690.1,
            "unit": "ns/op\t       5 B/op\t       0 allocs/op",
            "extra": "1707085 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x5_CacheRead20k - ns/op",
            "value": 690.1,
            "unit": "ns/op",
            "extra": "1707085 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x5_CacheRead20k - B/op",
            "value": 5,
            "unit": "B/op",
            "extra": "1707085 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x5_CacheRead20k - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "1707085 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x10_CacheRead20k",
            "value": 1651,
            "unit": "ns/op\t      11 B/op\t       0 allocs/op",
            "extra": "733848 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x10_CacheRead20k - ns/op",
            "value": 1651,
            "unit": "ns/op",
            "extra": "733848 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x10_CacheRead20k - B/op",
            "value": 11,
            "unit": "B/op",
            "extra": "733848 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x10_CacheRead20k - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "733848 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x20_CacheRead20k",
            "value": 3894,
            "unit": "ns/op\t      27 B/op\t       0 allocs/op",
            "extra": "307033 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x20_CacheRead20k - ns/op",
            "value": 3894,
            "unit": "ns/op",
            "extra": "307033 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x20_CacheRead20k - B/op",
            "value": 27,
            "unit": "B/op",
            "extra": "307033 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x20_CacheRead20k - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "307033 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "pilotso11",
            "username": "pilotso11"
          },
          "committer": {
            "name": "pilotso11",
            "username": "pilotso11"
          },
          "id": "7324c7ad25e90223e730ffe1864744a14c9cc8fe",
          "message": "codebase-improvements",
          "timestamp": "2025-06-08T10:13:44Z",
          "url": "https://github.com/pilotso11/lazywritercache/pull/11/commits/7324c7ad25e90223e730ffe1864744a14c9cc8fe"
        },
        "date": 1749398597706,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkCacheWriteMax20kLF",
            "value": 347,
            "unit": "ns/op\t     109 B/op\t       6 allocs/op",
            "extra": "3408241 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax20kLF - ns/op",
            "value": 347,
            "unit": "ns/op",
            "extra": "3408241 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax20kLF - B/op",
            "value": 109,
            "unit": "B/op",
            "extra": "3408241 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax20kLF - allocs/op",
            "value": 6,
            "unit": "allocs/op",
            "extra": "3408241 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax100kLF",
            "value": 402.8,
            "unit": "ns/op\t     116 B/op\t       7 allocs/op",
            "extra": "2591595 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax100kLF - ns/op",
            "value": 402.8,
            "unit": "ns/op",
            "extra": "2591595 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax100kLF - B/op",
            "value": 116,
            "unit": "B/op",
            "extra": "2591595 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax100kLF - allocs/op",
            "value": 7,
            "unit": "allocs/op",
            "extra": "2591595 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead20kLF",
            "value": 73.61,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "16103270 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead20kLF - ns/op",
            "value": 73.61,
            "unit": "ns/op",
            "extra": "16103270 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead20kLF - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "16103270 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead20kLF - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "16103270 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead100kLF",
            "value": 127,
            "unit": "ns/op\t       4 B/op\t       0 allocs/op",
            "extra": "8540643 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead100kLF - ns/op",
            "value": 127,
            "unit": "ns/op",
            "extra": "8540643 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead100kLF - B/op",
            "value": 4,
            "unit": "B/op",
            "extra": "8540643 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead100kLF - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "8540643 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x5_CacheRead20kLF",
            "value": 158.2,
            "unit": "ns/op\t       1 B/op\t       0 allocs/op",
            "extra": "7587050 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x5_CacheRead20kLF - ns/op",
            "value": 158.2,
            "unit": "ns/op",
            "extra": "7587050 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x5_CacheRead20kLF - B/op",
            "value": 1,
            "unit": "B/op",
            "extra": "7587050 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x5_CacheRead20kLF - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "7587050 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x10_CacheRead20kLF",
            "value": 319.5,
            "unit": "ns/op\t       2 B/op\t       0 allocs/op",
            "extra": "3397314 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x10_CacheRead20kLF - ns/op",
            "value": 319.5,
            "unit": "ns/op",
            "extra": "3397314 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x10_CacheRead20kLF - B/op",
            "value": 2,
            "unit": "B/op",
            "extra": "3397314 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x10_CacheRead20kLF - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "3397314 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x20_CacheRead20kLF",
            "value": 631,
            "unit": "ns/op\t       5 B/op\t       0 allocs/op",
            "extra": "1691977 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x20_CacheRead20kLF - ns/op",
            "value": 631,
            "unit": "ns/op",
            "extra": "1691977 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x20_CacheRead20kLF - B/op",
            "value": 5,
            "unit": "B/op",
            "extra": "1691977 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x20_CacheRead20kLF - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "1691977 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax20k",
            "value": 269.1,
            "unit": "ns/op\t     150 B/op\t       3 allocs/op",
            "extra": "4350591 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax20k - ns/op",
            "value": 269.1,
            "unit": "ns/op",
            "extra": "4350591 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax20k - B/op",
            "value": 150,
            "unit": "B/op",
            "extra": "4350591 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax20k - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "4350591 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax100k",
            "value": 287,
            "unit": "ns/op\t     144 B/op\t       3 allocs/op",
            "extra": "3875474 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax100k - ns/op",
            "value": 287,
            "unit": "ns/op",
            "extra": "3875474 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax100k - B/op",
            "value": 144,
            "unit": "B/op",
            "extra": "3875474 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheWriteMax100k - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "3875474 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead20k",
            "value": 79.69,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "13448524 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead20k - ns/op",
            "value": 79.69,
            "unit": "ns/op",
            "extra": "13448524 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead20k - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "13448524 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead20k - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "13448524 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead100k",
            "value": 106.9,
            "unit": "ns/op\t       3 B/op\t       0 allocs/op",
            "extra": "10265738 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead100k - ns/op",
            "value": 106.9,
            "unit": "ns/op",
            "extra": "10265738 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead100k - B/op",
            "value": 3,
            "unit": "B/op",
            "extra": "10265738 times\n4 procs"
          },
          {
            "name": "BenchmarkCacheRead100k - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "10265738 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x5_CacheRead20k",
            "value": 689.5,
            "unit": "ns/op\t       4 B/op\t       0 allocs/op",
            "extra": "1731213 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x5_CacheRead20k - ns/op",
            "value": 689.5,
            "unit": "ns/op",
            "extra": "1731213 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x5_CacheRead20k - B/op",
            "value": 4,
            "unit": "B/op",
            "extra": "1731213 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x5_CacheRead20k - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "1731213 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x10_CacheRead20k",
            "value": 1656,
            "unit": "ns/op\t      11 B/op\t       0 allocs/op",
            "extra": "729040 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x10_CacheRead20k - ns/op",
            "value": 1656,
            "unit": "ns/op",
            "extra": "729040 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x10_CacheRead20k - B/op",
            "value": 11,
            "unit": "B/op",
            "extra": "729040 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x10_CacheRead20k - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "729040 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x20_CacheRead20k",
            "value": 3905,
            "unit": "ns/op\t      27 B/op\t       0 allocs/op",
            "extra": "309464 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x20_CacheRead20k - ns/op",
            "value": 3905,
            "unit": "ns/op",
            "extra": "309464 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x20_CacheRead20k - B/op",
            "value": 27,
            "unit": "B/op",
            "extra": "309464 times\n4 procs"
          },
          {
            "name": "BenchmarkParallel_x20_CacheRead20k - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "309464 times\n4 procs"
          }
        ]
      }
    ]
  }
}
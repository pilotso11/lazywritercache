window.BENCHMARK_DATA = {
  "lastUpdate": 1749376815768,
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
      }
    ]
  }
}
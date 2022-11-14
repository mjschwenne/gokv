#!/usr/bin/env python3

import matplotlib.pyplot as plt
import json

def plot(filename, title=None):
    xs = []
    ys = []
    with open('data/' + filename, newline='') as f:
        for l in f.readlines():
            if l.startswith("#"):
                continue
            o = json.loads(l)
            xs.append(o['lts']['UPDATE']['thruput'])
            ys.append(o['lts']['UPDATE']['avg_latency']/1000)

    plt.plot(xs, ys, marker='o', label=title)
    plt.ylabel('latency (ms)')
    plt.xlabel('throughput')

# plot('hdd/pb-kvs.jsons', 'hdd-pb')
# plot('hdd/redis-kvs.jsons', 'hdd-redis')
# plt.legend()
# plt.show()

# plot('ssd/pb-kvs.jsons', 'ssd-pb')
# plot('ssd/redis-kvs.jsons', 'ssd-redis')
# plot('ssd/pb-kvs2.jsons', 'ssd-pb2')
# plt.legend()
# plt.show()

plot('hdd/pb-kvs-11-07.jsons', 'ssd-pb')
plot('hdd/redis-kvs-11-07.jsons', 'ssd-redis')
plt.legend()
plt.show()

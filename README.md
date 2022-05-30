## Timebased
This repo contains useful methods can be used in your daily development.

### Timebased stats for a period of time.
You want to know most viewed products within 3 hours. Assume we have 3 buckets to store recent 3 hours views into them, here is a list

| Hour 1:00 | Hour 2:00 | Hour 3:00 |
| - | - | - |
|P1: 0|P1: 13|p1: 9|
|P2: 5|P2: 11|p2: 29|
|P3: 7|P3: 0|p3: 0|
|P4: 0|P4: 0|p4: 21|

Now, we are at fourth hour, (assume it's 3:10), we can get a rank list at the end.

1. P2: 45
1. p1: 22
1. p4: 21
1. p3: 7

You can check the implementation under `examples/`.

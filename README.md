## Timebased
This repo contains useful methods can be used in your daily development.

### Timebased stats for a period of time.
You want to know most viewed products within 3 hours. Assume we have 3 buckets to store recent 3 hours views into them, here is a list

| Hour One | Hour Two | Hour Three |
| - | - | - |
|P1: 0|P1: 13|p1: 9|
|P2: 5|P2: 11|p2: 29|
|P3: 7|P3: 0|p3: 0|
|P4: 0|P4: 0|p4: 21|

Now, we are at fourth hour, (assume it's 3:10), we can report a rank list.

1. P2: 45
2. p1: 22
2. p4: 21
2. p3: 7

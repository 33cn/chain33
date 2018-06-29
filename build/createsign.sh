#!/usr/bin/env bash
keylist=(1FB8L3DykVF7Y78bRfUrRcMZwesKue7CyR 1Lw6QLShKVbKM6QvMaCQwTh5Uhmy4644CG 1PSYYfCbtSeT1vJTvSKmQvhz8y6VhtddWi 1BG9ZoKtgU5bhKLpcsrncZ6xdzFCgjrZud 1G7s64AgX1ySDcUdSW5vDa8jTYQMnZktCd 1FiDC6XWHLe7fDMhof8wJ3dty24f6aKKjK 1AMvuuQ7V7FPQ4hkvHQdgNWy8wVL4d4hmp 1ExRRLoJXa8LzXdNxnJvBkVNZpVw3QWMi4 1KNGHukhbBnbWWnMYxu1C7YMoCj45Z3amm 1AH9HRd4WBJ824h9PP1jYpvRZ4BSA4oN6Y)
i=0
while IFS='' read -r line || [[ -n $line ]]; do
    ./chain33-cli send bty transfer -k 0x1E897C63904A1F943BCC39CB65F0F30C65E395D8B571AF66A08DF8E8408969E3 -n "挖矿手续费" -a 100 -t "${keylist[$i]}"
    ((i++))
done <"$1"

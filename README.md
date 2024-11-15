# iptraf-fup
Internet volume monitor and disconnector based on iptraf tool
It requires to run on HotSpot Linux PC/Server 

Linux x86 Binary:
https://github.com/motaz/iptraf-fup/releases/download/v0.9.0/iptraf.tar.xz

After install iptraf tool, run it in background using below command in root user:

/usr/sbin/iptraf -l all -f -B -L /var/log/traffic.log 

After than you can run iptraf-fup in root crontab every 5 minutes

Configure PC/Server WIFI Link MAC in config.ini in skiplist parameter to prevent disconnecting server MAC

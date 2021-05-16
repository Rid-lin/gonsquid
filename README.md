# Minimalist Netflow v5 to squid-log collector written in Go

The broker listens on UDP port (default 2055), accepts Netflow traffic, and by default collects records with selected metadata formatted into squid log. Login information replaces the Mac address of the device that receives from the router mikrotik.

To build the report, it uses the [screensquid](https://sourceforge.net/projects/screen-squid/) database and its part (fetch.pl) for parsing and loading the squid log into the database

## Usage

Clone repository

`git clone https://github.com/Rid-lin/gonsquid.git`

`cd gonsquid`

Copy folder assets to /usr/share/gonsquid/

`cp /assets /usr/share/gonsquid/`

Build programm:

`make build`

Move binary file

`mv ./bin/linux/gonsquid /usr/local/bin/`

Edit file /usr/share/gonsquid/assets/gonsquid.service

`nano /usr/share/gonsquid/assets/gonsquid.service`

E.g.

`/usr/local/bin/gonsquid -subnet=10.0.0.0/8 -subnet=192.168.0.0/16 -ignorlist=10.0.0.2 -ignorlist=:3128 -ignorlist=8.8.8.8:53 -ignorlist=ff02:: -loglevel=debug -log=/var/log/gonsquid/access.log -mtaddr=192.168.1.1:8728 -u=mikrotik_user -p=mikrotik_user_password`

and move to /lib/systemd/system

`mv /usr/share/gonsquid/assets/gonsquid.service /lib/systemd/system`

Make sure the log folder exists, If not then

`mkdir -p /var/log/gonsquid/`

Configuring sistemd to automatically start the program

`systemctl daemon-reload`

`systemctl start gonsquid`

`systemctl enable gonsquid`

Edit the file "fetch.pl" in accordance with the comments and recommendations.

Add a task to cron (start every 5 minutes)

`crontab -e`

`*/5 * * * * /var/www/screensquid/fetch.pl > /dev/null 2>&1`

## Supported command line parameters

```
Usage of gonsquid.exe:
  -bind_addr string
        Listen address for response mac-address from mikrotik (default ":3030")
  -csv string
        Output to csv (default "false")
  -default_quota_daily string
        Default daily traffic consumption quota (default "0")
  -default_quota_hourly string
        Default hourly traffic consumption quota (default "0")
  -default_quota_monthly string
        Default monthly traffic consumption quota (default "0")
  -flow_addr string
        Address and port to listen NetFlow packets (default "0.0.0.0:2055")
  -ignor_list string
        List of lines that will be excluded from the final log
  -interval string
        Interval to getting info from Mikrotik (default "10m")
  -loc string
        Location for time (default "Asia/Yekaterinburg")
  -log_level string
        Log level: panic, fatal, error, warn, info, debug, trace (default "info")
  -mt_addr string
        The address of the Mikrotik router, from which the data on the comparison of the MAC address and IP address is taken
  -mt_pass string
        The password of the user of the Mikrotik router, from which the data on the comparison of the mac-address and IP-address is taken      
  -mt_user string
        User of the Mikrotik router, from which the data on the comparison of the MAC address and IP address is taken
  -name_file_to_log string
        The file where logs will be written in the format of squid logs
  -num_of_trying_connect_to_mt string
        The number of attempts to connect to the microtik router (default "10")
  -receive_buffer_size_bytes string
        Size of RxQueue, i.e. value for SO_RCVBUF in bytes
  -size_one_megabyte string
        The number of bytes in one megabyte (default "1048576")
  -sub_nets string
        List of subnets traffic between which will not be counted
  -use_tls string
        Using TLS to connect to a router (default "false")
```

## Credits

This project was created with help of:

* https://github.com/strzinek/gonflux
* https://sourceforge.net/projects/screen-squid/
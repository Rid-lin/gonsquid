# Minimalist Netflow v5 to squid-log and CSV collector written in Go

The broker listens on UDP port (default 2055), accepts Netflow v5 traffic, and by default collects records with selected metadata formatted into squid log. Login information replaces the Mac address of the device that receives from the router mikrotik.

Any squid log analyzer can be used to generate reports. For example: [screensquid](https://sourceforge.net/projects/screen-squid/)

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

```bash
Usage:
  -bind_addr string
        Listen address for HTTP-server (default "0.0.0.0:30340")
  -csv string
        Output to a CSV file, equivalent to the setting for squid 4.0+ in squid.conf 'logformat csv %{%Y|%b|%d|%H|%M|%S|%z}tl|%tr|%st|%>a|%>A|%>p|%>eui|%<a|%<p|%ru|%Ss|%03>Hs|%rm|%[un|%Sh/%<a|%mt' (default "false")
  -flow_addr string
        Address and port to listen NetFlow packets (default "0.0.0.0:2055")
  -gomtc_addr string
        Address and port for connect to GOMTC API (default "http://127.0.0.1:3034")
  -ignor_list string
        List of lines that will be excluded from the final log
  -interval string
        Interval to getting info from GOMTC (default "1m")
  -loc string
        Location for time (default "Asia/Yekaterinburg")
  -log_level string
        Log level: panic, fatal, error, warn, info, debug, trace (default "info")
  -name_file_to_log string
        The file where logs will be written in the format of squid logs (default "access.log")
  -receive_buffer_size_bytes string
        Size of RxQueue, i.e. value for SO_RCVBUF in bytes
  -sub_nets string
```

## Credits

This project was created with help of:

* <https://github.com/strzinek/gonflux>
* <https://sourceforge.net/projects/screen-squid/>

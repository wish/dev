#
# Regular cron jobs for the dev package
#
0 4	* * *	root	[ -x /usr/bin/dev_maintenance ] && /usr/bin/dev_maintenance

CREATE TABLE `servers` (
	`id`	INTEGER PRIMARY KEY AUTOINCREMENT,
	`hostname`	TEXT,
	`ip`	TEXT,
	`enablehttp`	INTEGER DEFAULT 0,
	`httpresult`	TEXT,
	`enablestmp`	INTEGER DEFAULT 0,
	`smtpresult`	TEXT,
	`smtpport`	INTEGER DEFAULT 25,
	`enablepop3`	INTEGER DEFAULT 0,
	`pop3result`	TEXT,
	`enablehttps`	INTEGER DEFAULT 0,
	`httpsresult`	TEXT,
	`enableping`	INTEGER DEFAULT 0,
	`pingresult`	TEXT
);
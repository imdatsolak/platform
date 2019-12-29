drop database if exists cygnusa;

create database cygnusa;
grant all on cygnusa.* to 'cygnusa' identified by 'cygnusa';
use cygnusa;

CREATE TABLE Accounts (
    accountId INT(10) NOT NULL PRIMARY KEY AUTO_INCREMENT,
    accountName VARCHAR(64) DEFAULT NULL,
    login VARCHAR(32) NOT NULL DEFAULT '',
    password VARCHAR(32) DEFAULT NULL,
    backupEmail VARCHAR(64) DEFAULT NULL,
    backupMobilePhone VARCHAR(32) DEFAULT NULL,
    notificationsEmail VARCHAR(64) DEFAULT NULL,
    notificationsPhone VARCHAR(32) DEFAULT NULL,
    disabled TINYINT NOT NULL DEFAULT 0,
    creationDate DATE DEFAULT NULL
);

CREATE TABLE Contacts (
    contactId INT(10) NOT NULL PRIMARY KEY AUTO_INCREMENT,
    accountId INT(10) NOT NULL DEFAULT 0,
    contactType INT NOT NULL DEFAULT 0,
    department VARCHAR(32) NOT NULL DEFAULT '',
    salutation VARCHAR(16) NOT NULL DEFAULT '',
    firstname VARCHAR(32) NOT NULL DEFAULT '',
    middlename VARCHAR(32) NOT NULL DEFAULT '',
    lastname VARCHAR(32) NOT NULL DEFAULT '',
    street1 VARCHAR(64) NOT NULL DEFAULT '',
    street2 VARCHAR(64) NOT NULL DEFAULT '',
    street3 VARCHAR(64) NOT NULL DEFAULT '',
    zip VARCHAR(16) NOT NULL DEFAULT '',
    city VARCHAR(64) NOT NULL DEFAULT '',
    state VARCHAR(32) NOT NULL DEFAULT '',
    country VARCHAR(2) NOT NULL DEFAULT 'de',
    phone1 VARCHAR(32) NOT NULL DEFAULT '',
    phone2 VARCHAR(32) NOT NULL DEFAULT '',
    email1 VARCHAR(64) NOT NULL DEFAULT '',
    email2 VARCHAR(64) NOT NULL DEFAULT '',
    vatid VARCHAR(16) NOT NULL DEFAULT '',
    notes VARCHAR(128) NOT NULL DEFAULT ''
);

CREATE TABLE Applications (
    applicationId INT(10) NOT NULL PRIMARY KEY AUTO_INCREMENT,
    accountId INT(10) NOT NULL DEFAULT 1,
    applicationName VARCHAR(32) NOT NULL DEFAULT '',
    applicationLogin VARCHAR(32) NOT NULL DEFAULT '',
    applicationSecret VARCHAR(32) NOT NULL DEFAULT '',
    userInfo VARCHAR(128) NOT NULL DEFAULT '',
    disabled TINYINT NOT NULL DEFAULT 0
);


CREATE TABLE ApplicationInstances (
    aInstanceId INT(10) NOT NULL PRIMARY KEY AUTO_INCREMENT,
    applicationId INT(10) NOT NULL DEFAULT 1,
    applicationInstanceUID VARCHAR(48) NOT NULL DEFAULT '',
    disabled TINYINT NOT NULL DEFAULT 0
);

CREATE TABLE DoneJobs (
    doneJobId INT(10) NOT NULL PRIMARY KEY AUTO_INCREMENT,
    tempJobId INT(10) NOT NULL DEFAULT 0,
    applicationId INT(10) NOT NULL DEFAULT 0,
    applicationInstanceId INT(10) NOT NULL DEFAULT 0,
    jobUID VARCHAR(48) NOT NULL DEFAULT '',
    requestType INT NOT NULL DEFAULT 0,
    requestStartTime DATETIME NULL,
    requestSize INT NOT NULL DEFAULT 0,
    requestData TEXT NULL,
    requestEndTime DATETIME NULL,
    processingTime INT(10) NOT NULL DEFAULT 0
);

CREATE TABLE TempJobs (
    jobId INT(10) NOT NULL PRIMARY KEY AUTO_INCREMENT,
    applicationId INT(10) NOT NULL DEFAULT 0,
    applicationInstanceId INT(10) NOT NULL DEFAULT 0,
    jobUID VARCHAR(48) NOT NULL DEFAULT '',
    jobStatus INT NOT NULL DEFAULT 0,
    requestType INT NOT NULL DEFAULT 0,
    requestStartTime DATETIME NULL,
    requestSize INT NOT NULL DEFAULT 0,
    requestData TEXT NULL,
    uploadId VARCHAR(48) NOT NULL DEFAULT '',
    requestEndTime DATETIME NULL,
    processingTime INT(10) NOT NULL DEFAULT 0,
    jobResultDataPtr VARCHAR(128) NOT NULL DEFAULT '',
    jobResultRetrieved INT NOT NULL DEFAULT 0,
    uploadIdentifier VARCHAR(64) NULL DEFAULT ''
);

CREATE TABLE AvailableServices (
    serviceId INT(10) NOT NULL PRIMARY KEY AUTO_INCREMENT,
    services TEXT NULL
);

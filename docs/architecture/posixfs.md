---
title: "Posix Filesystem"
date: 2024-05-27T14:31:00+01:00
weight: 30
geekdocRepo: https://github.com/owncloud/ocis
geekdocEditPath: edit/master/docs/architecture
geekdocFilePath: posixfs.md
---

{{< toc >}}

Posix FS is the working name for the collaborative storage driver for Infinite Scale.

The scope of this document is to give an high level overview to the technical aspects of the Posix FS and guide the setup.

## Introduction

Posix FS is a backend component that manages files on the server utilizing a "real" file tree that represents the data with folders and files in the file system as users are used to it. That is the big difference compared to Decomposed FS which is the default storage driver in Infinite Scale.

This does not mean that Infinte Scale is trading any of it's benefits to this new feature: It still implements simplicity by running without a database, it continues to store metadata in the file system and adds them transparently to chaches and search index, and it also features the full spaces concept as before, just to name a few example.

The architecture of Infinite Scale allows to configure different storage drivers for specific storage types and purposes on a space granularity. The Posix FS storage driver is an alternative to the default driver called Decomposed FS.

However, the clearance of the file structure in the underlying file system is not the only benefit of the Posix FS. This new technology allows users to manipulate the data directly in the file system, and any changes to files made aside of Infinite Scale are monitored and directly reflected in Infinite Scale.

The first time ever with feature rich open source file synce & share systems, users can either choose to work with their data through the clients of the system, it's API's or even directly in the underlying file system on the server.

That is another powerful vector for integration and enables a new spectrum of use cases across all domains. Just imagine how many software can write files, and can now directly make them accessible real time in a convenient, secure and efficient way.

## Technical Aspects

The Posix FS technology uses a few features of the underlying file system, which are mandatory and directly contributing to the performance of the system.

While the simplest form of Posix FS runs with default file systems of every modern Linux system, the full power of this unfolds with more capable file systems such as IBM Storage Scale or Ceph. These are recommended as reliable foundations for big installations of Infinite Scale.

This chapter describes some technical aspects of the storage driver.

### Path Locations

The file tree that is used as storage path for both data and metadata is located under the local path on the machine that is running Infinite Scale. That might either be a real local file system or a mounted net filesystem. It is expected that oCIS is the only consumer of that file tree, except what is expected behaviour with a collaborative file system, that works with files in that tree.

Underneath the Infinite Scale file system root, there is a collection of different folders containing Infinite Scale specific data storing personal spaces, project spaces and indexes.

### Metadata

Infinite Scale is highly dependent on the efficient usage of meta data which are attached to file resources, but also logical elements such as spaces.

Metadata are stored in extended attributes (as also supported by decompsed FS) which poses the benefit that metadata is always directly attached to the actual resources. As a result care has to be taken that extended attributes are considered when working with the file tree however, e.g. when creating or restoring backups.

Note: The maximum number and size of extended attributes are limited depending on the filesystem and block size. See [GPFS Specifics](#gpfs-specifics) for more details on GPFS file systems.

All indexing and caching of metadata is implemented in higher system levels than the storage driver, and thus are not different to the components used with other storage drivers like the decomposed FS.

### Monitoring

To get information about changes such as new files added, files edited or removed, Infinte Sale uses a monitoring system to directly watch the file system. This starts with the Linux inotify system and ranges to much more sophisticated services as for example in Spectrum Scale (see [GPFS Specifics](#gpfs-specifics) for more details on GPFS file systems).

Based on the information transmitted by the watching service, Infinite Scale is able to "register" new or changed files into its own caches and internal management structures. This enables Infinite Scale to deliver resource changes through the "traditional" channels such as APIs and clients.

Since the most important metadata is the file tree structure itself, the "split brain" situation between data and metadata is impossible to cause trouble.

### Automatic ETag Propagation

The ETag of a resource can be understood as a content fingerprint of any file- or folder resource in Infinite Scale. It is mainly used by clients to detect changes of resources. The rule is that if the content of a file changed the ETag has to change as well, as well as the ETag of all parent folders up to the root of the space.

Infinite Scale uses a built in mechanism to maintain the ETag for each resource in the file meta data, and also propagates it automatically.

In the future a sophisticated underlying file system could provide an attribute that fulfills this requirement and changes whenever content or metadata of a resource changes, and - which is most important - also changes the attribute of the parent resource and the parent of the parent etc.

### Automatic Tree Size Propagation

Similar to the ETag propagation described before, Infinite Scale also tracks the accumulated tree size in all nodes of the file tree. A change to any file requires a re-calculation of the size attribute in all parent folders.

In the future Infinite Scale could benefit from file systems with native tree size propagation.

### Quota

Each space has it's own quota, thus every storage driver implementation needs to consider that.

For example, IBM Spectrum Scale supports quota handling directly in the file system.

Other systems store quota data in the metadata storage and implement propagation of used quota similar to the tree size propagation.

### File Id Resolution

Infinite Scale uses an Id based approach to work with resources, rather than a file path based mechanism. The reason for that is that Id based lookups can be done way more efficient compared to tree traversals, just to name one reason.

The most important component of the Id is a unique file Id that identifies the resource within a space. Ideally the Inode of a file could be used here. However, some file systems re-use inodes which must be avoided. Infinite Scale thus does not use the file Inode, but generates a UUID instead.

ID based lookups utilize an Id cache which needs to be shared between all storageprovider and dataprovider instances. During startup a scan of the whole file tree is performed to detect and cache new entities.

In the future a powerful underlying file system could support Infinite Scale by providing an API that

1. Provides the Id for a given file path referenced resource
2. Provides the path for a given Id.

These two operations are very crucial for the performance of the entire system.

### User Management

With the requirement that data can be manipulated either through the filesystem or the Infinite Scale system, the question under which uid the manipulation happens is an important question.

There are a few possible ways for user management:
1. Changes can either be only accepted by the same  user that Infinite Scale is running under, for example the user `ocis`. All manipulations in the filesystem have to be done by, and only by, this user.
2. Group based: All users who should be able to manipulate files have to be in a unix group. The Infinite Scale user has also to be in there. The default umask in the directory used has to allow group writing all over the place.
3. Impersonation: Infinite Scale impersonates the user who owns the folder on the file system to mimic the access as the user.

All possibilities have pros and cons for operations.

One for all, it seems reasonable to use LDAP to manage users which is the base for the Infinite Scale IDP as well as the system login system via PAM.

### GID based space access

Posix FS supports GID based space access to support the problem that project spaces might have to be accessible by multiple users on disk. In order to enable this feature the `ocis` binary needs to have the `setgid` capability and `STORAGE_USERS_POSIX_USE_SPACE_GROUPS` needs to be set to `true`. Inifinite Scale will then use the space GID (the gid of the space root) for all file system access using the `setfsgid` syscall, i.e. all files and directories created by Infinite Scale will belong to the same group as the space root.

## Advanced Features

Depending on the capabilities of the underlying file system, the Infinite Scale Posix FS can benefit from more advanced functionality described here.

### Versioning

If the underlying file system is able to create versions of single resources (imagine a git based file system) this functionality could directly be used by Infinite Scale.

In the current state of the PosixFS, versioning is not supported.

### Trashbin

If the underlying file system handles deleted files in a trash bin that allows restoring of previously removed files, this functionality could directly be used by Infinite Scale.

If not available it will follow the [the Free Desktop Trash specificaton](https://specifications.freedesktop.org/trash-spec/trashspec-latest.html).

In the current state of the PosixFS, trash bin is not supported.

## Limitations

As of Q2/2024 the PosixFS is in technical preview state which means that it is not officially supported.

The tech preview comes with the following limitations:

1. Only inotify and GPFS file system change notification methods are supported
1. Advanced features versioning and trashbin are not supported yet
1. The space/project folders in the filesystem are named after the UUID, not the real space name
1. No CephFS support yet

## Setup

This describes the steps to use the PosixFS storage driver with Infinite Scale.

It is possible to use different storage drivers in the same Infinite Scale installation. For example it is possible to set up one space running on PosixFS while others run decomposedFS.

### Prerequisites

To run PosixFS, the following prerequisites have to be fulfilled:

1. There must be storage available to store meta data and blobs, available under a root path
1. When using inotify, the storage must be local on the same machine. Network mounts do not work with inotify. `inotifywait` needs to be installed.
1. The storage root path must be writeable and executable by the same user Infinite Scale is running under
1. An appropiate version of Infinite Scale is installed, version number 5.0.5 and later
1. Either redis or nats-js-kv cache service


### Setup Configuration

This is an example configuration with environment variables that configures Infinite Scale to use PosixFS for all spaces it works with, ie. Personal and Project Spaces:

```
export STORAGE_USERS_DRIVER="posix"
export STORAGE_USERS_POSIX_ROOT="/home/kf/tmp/posix-storage"
export STORAGE_USERS_POSIX_WATCH_TYPE="inotifywait"
export STORAGE_USERS_ID_CACHE_STORE="nats-js-kv"            # for redis "redis"
export STORAGE_USERS_ID_CACHE_STORE_NODES="localhost:9233"  # for redis "127.0.0.1:6379"
export STORAGE_USERS_POSIX_USE_SPACE_GROUPS="true"          # optional. Enable when using gid based space access
```

## GPFS Specifics

When using GPFS as the underlying filesystem the machine running the according `storage-users` service needs to have the GPFS filesystem mounted locally. The mount path is given to ocis as the `STORAGE_USERS_POSIX_ROOT` path.

Other than that there a few other points to consider:

### Extended Attributes

As described above metadata is stored as extended attributes of the according entities and thus is suspect to their limitations. In GPFS extended attributes are first stored in the inode itself but can then also use an overflow block which is at least 64KB and up to the metadata block size. Inode and metadata block size should be chosen accordingly.

### FS Watcher

Posix FS supports two different watchers for detecting changes to the filesystem. The watchfolder watcher is better tested and supported at that point.

#### GPFS File Audit Logging

The gpfsfileauditlogging watcher tails a GPFS file audit log and parses the JSON events to detect relevant changes.

```
export STORAGE_USERS_POSIX_WATCH_TYPE="gpfsfileauditlogging"
export STORAGE_USERS_POSIX_WATCH_PATH="/path/to/current/audit/log"
```

#### GPFS Watchfolder

The gpfswatchfolder watcher connects to a kafka cluster which is being filled with filesystem events by the GPFS watchfolder service.

```
export STORAGE_USERS_POSIX_WATCH_TYPE="gpfswatchfolder"
export STORAGE_USERS_POSIX_WATCH_PATH="fs1_audit"                               # the kafka topic to watch
export STORAGE_USERS_POSIX_WATCH_FOLDER_KAFKA_BROKERS="192.168.1.180:29092"
```
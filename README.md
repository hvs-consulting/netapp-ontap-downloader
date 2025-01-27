# NetApp ONTAP downloader

NetApp ONTAP storages provide a powerful REST API, that allows to configure the storage system and perform file accesses (provided that valid credentials with sufficient permissions are available).
This tool downloads a file via the NetApp ONTAP REST API and stores it locally.
The challenge is that the maximal amount of data that can be retrieved with one API call is limited - this tool downloads the file in 1MB chunks.

## Usage

Build:

    go build

To download a file, the volume ID, path and of course URL of the storage and credentials are required:

    ./downloader -url "https://storage.internal" -user "admin" -password "hopefully-not-admin" -volume-id "11111111-2222-3333-4444-555555555555" -file-path "vms/dc01/dc01-flat.vmdk" -output dc01-flat.vmdk

Optional parameters:
    
* `-skip-tls`: Do not verify the TLS certificate of the target
* `-chunk-size 1000000`: Change the chunk size in bytes (default: 1000000 - 1MB)

## Why?

During one of our engagements, we obtained administrative access to a NetApp storage, that contained disks of virtual domain controllers.
However, we were not able to access these files without configuration changes (like changing the NFS configuration).
Since that was not an viable option, we explored the REST API and implemented this little tool.

## Details

The used API endpoint is (see also [NetApp documentation](https://docs.netapp.com/us-en/ontap-restapi/ontap/get-storage-volumes-files-.html)):

    GET /storage/volumes/{volume.uuid}/files/{path}

In case you need to find the volume ID and file names via API, these API endpoints are useful:

* `GET /api/storage/volumes/`: Lists the volumes, so you can get the IDs
* `GET /api/storage/volumes/<volume id>/files/`: Lists the files in that volume
* `GET /api/storage/volumes/<volume id>/files/<path>`: List the files in that path (path needs to be URL encoded) - this API endpoint is then also used for the download.

The tool was tested on NetApp Release 9.11.1P14.

We used this tool to download multiple files between 100 and 200 GB.
The API has proven to be solid and reliable.

## Further notes

This is not a vulnerability and requires administrative access to the storage.
With administrative access, we could simply change the configuration of the storage - but in our situation this was not desired.

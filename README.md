```bash
 make docker
 ```

 ```bash
 kind load docker-image minioupgrade:local --name kind
 ```

 ### POST request to create binaries

 POST http://upgrademinio-service.ns-1.svc.cluster.local:3000/create
 ```json
 {
     "imagename": "minio:latest"
 }
 ```

 #### Response example

 ```json
 {
   "minio": "minio.RELEASE.2024-10-13T13-34-11Z",
   "MinioSha256": "minio.RELEASE.2024-10-13T13-34-11Z.sha256sum",
   "minisig": "minio.RELEASE.2024-10-13T13-34-11Z.minisig"
 }
 ```

 ```bash
 mc admin update cluster http://upgrademinio-service.ns-1.svc.cluster.local:3000/RELEASE.2024-10-13T13-34-11Z/minio.RELEASE.2024-10-13T13-34-11Z.sha256sum --insecure
 ```

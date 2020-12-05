#!/bin/bash
cd "$(dirname "$0")"
dev_appserver.py --default_gcs_bucket_name gopherizeme.appspot.com ./app.yaml

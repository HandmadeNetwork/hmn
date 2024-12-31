#!/usr/bin/env python3

import os

import boto3

# You must already have configured your "AWS" (DigitalOcean) credentials via the AWS CLI.

s3 = boto3.resource("s3")
bucket = s3.Bucket("hmn-backup")
for obj in bucket.objects.filter(Prefix="db"):
    print(obj.key)

print()
print("Above is a list of all the available database backups.")
print("Enter the name of the one you would like to download (e.g. \"hmn_pg_dump_live_2023-09-24\"):")
filename = input()

s3 = boto3.client("s3")
s3.download_file("hmn-backup", f"db/{filename}", os.path.join("local", "backups", filename))

print(f"Downloaded {filename} to local/backups.")

#!/usr/bin/env sh

# Make html style assets available to the DOI hosting service.
# When the '/doidata' volume is mounted from the host on startup
# of the docker container, any data in this directory is no longer
# available.
# This script copies all required assets to the '/doidata' volume
# after startup, making the assets available to the DOI hosting service.
# If required also creates the '/doidata/assets' directory.
echo "Preparing 'assets' directory; overwriting any previous 'assets' changes."
mkdir -p /doidata/assets
cp -vr /assets/* /doidata/assets

# Start DOI registration server
echo "Starting DOI registration server"
/gindoid start

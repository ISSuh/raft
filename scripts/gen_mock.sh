#!/bin/sh

pushd ./internal
mockery --all --inpackage
popd

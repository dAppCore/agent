#!/bin/bash
set -e

core php artisan migrate:rollback

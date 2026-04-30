<?php

use Illuminate\Support\Facades\Route;

Route::apiResource('users', 'UserController');
Route::post('auth/login', ['AuthController', 'login']);

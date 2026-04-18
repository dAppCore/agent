<?php

declare(strict_types=1);

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Add IP whitelist restrictions to agent_api_keys table.
     */
    public function up(): void
    {
        if (! Schema::hasTable('agent_api_keys')) {
            return;
        }

        Schema::table('agent_api_keys', function (Blueprint $table) {
            if (! Schema::hasColumn('agent_api_keys', 'ip_restriction_enabled')) {
                $table->boolean('ip_restriction_enabled')->default(false);
            }
            if (! Schema::hasColumn('agent_api_keys', 'ip_whitelist')) {
                $table->json('ip_whitelist')->nullable();
            }
            if (! Schema::hasColumn('agent_api_keys', 'last_used_ip')) {
                $table->string('last_used_ip', 45)->nullable();
            }
        });
    }

    public function down(): void
    {
        if (! Schema::hasTable('agent_api_keys')) {
            return;
        }

        Schema::table('agent_api_keys', function (Blueprint $table) {
            $cols = [];
            if (Schema::hasColumn('agent_api_keys', 'ip_restriction_enabled')) {
                $cols[] = 'ip_restriction_enabled';
            }
            if (Schema::hasColumn('agent_api_keys', 'ip_whitelist')) {
                $cols[] = 'ip_whitelist';
            }
            if (Schema::hasColumn('agent_api_keys', 'last_used_ip')) {
                $cols[] = 'last_used_ip';
            }
            if ($cols) {
                $table->dropColumn($cols);
            }
        });
    }
};

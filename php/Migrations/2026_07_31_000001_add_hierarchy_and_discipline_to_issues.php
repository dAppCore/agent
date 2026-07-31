<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Give issues a parent, a discipline, and a repository.
     *
     * An epic is long-haul work that no single person finishes in one sitting;
     * it closes only once a spread of separate issues — planning, code, content,
     * analysis, creative — have each closed on their own terms. That needs three
     * things the table did not have: somewhere to hang a child (parent_id), a
     * way to say which discipline the child belongs to (discipline), and a way
     * to say which of several hundred repositories it concerns (repository).
     *
     * Guarded with hasColumn() so it is idempotent alongside the consolidated
     * app-level migration, matching create_issue_tracker_tables.
     */
    public function up(): void
    {
        if (! Schema::hasTable('issues')) {
            return;
        }

        Schema::table('issues', function (Blueprint $table) {
            if (! Schema::hasColumn('issues', 'parent_id')) {
                // nullOnDelete, not cascade: losing a parent should orphan the
                // children up to the top level, never silently delete work that
                // was scoped, estimated and possibly already part-done.
                $table->foreignId('parent_id')
                    ->nullable()
                    ->after('sprint_id')
                    ->constrained('issues')
                    ->nullOnDelete();
            }

            if (! Schema::hasColumn('issues', 'discipline')) {
                $table->string('discipline', 32)->nullable()->after('type');
            }

            if (! Schema::hasColumn('issues', 'repository')) {
                // A plain string ("dappcore/go-inference"), not a foreign key:
                // the repository register lives in the consuming app, not in
                // this package, and assignee/reporter already set the precedent.
                $table->string('repository')->nullable()->after('assignee');
            }
        });

        Schema::table('issues', function (Blueprint $table) {
            // Listing an epic's children, and filtering a workspace by
            // discipline or repository, are the three reads the board makes on
            // every page load.
            $table->index(['workspace_id', 'parent_id']);
            $table->index(['workspace_id', 'discipline']);
            $table->index(['workspace_id', 'repository']);
        });
    }

    public function down(): void
    {
        if (! Schema::hasTable('issues')) {
            return;
        }

        Schema::table('issues', function (Blueprint $table) {
            $table->dropIndex(['workspace_id', 'parent_id']);
            $table->dropIndex(['workspace_id', 'discipline']);
            $table->dropIndex(['workspace_id', 'repository']);
        });

        Schema::table('issues', function (Blueprint $table) {
            if (Schema::hasColumn('issues', 'parent_id')) {
                $table->dropConstrainedForeignId('parent_id');
            }

            foreach (['discipline', 'repository'] as $column) {
                if (Schema::hasColumn('issues', $column)) {
                    $table->dropColumn($column);
                }
            }
        });
    }
};

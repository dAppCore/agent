<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Mod\Admin\Menu\Contracts;

interface AdminMenuProvider
{
    /**
     * @return array<int, array{
     *     group: string,
     *     priority: int,
     *     admin?: bool,
     *     item: \Closure
     * }>
     */
    public function adminMenuItems(): array;

    /**
     * @return array<int, string>
     */
    public function menuPermissions(): array;

    public function canViewMenu(?object $user, ?object $workspace): bool;
}

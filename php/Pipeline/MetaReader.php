<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Pipeline;

interface MetaReader
{
    public function getPRMeta(int $prNumber): PRMeta;

    public function getEpicMeta(int $issueNumber): EpicMeta;

    public function getIssueState(int $issueNumber): IssueState;

    public function getCommentReactions(int $issueNumber, int $commentNumber): Reactions;
}

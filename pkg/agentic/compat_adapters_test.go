// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"

	store "dappco.re/go/store"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (s *PrepSubsystem) dispatch(ctx context.Context, request *mcp.CallToolRequest, input DispatchInput) (*mcp.CallToolResult, DispatchOutput, error) {
	return dispatch(s, ctx, request, input)
}

func (s *PrepSubsystem) watch(ctx context.Context, request *mcp.CallToolRequest, input WatchInput) (*mcp.CallToolResult, WatchOutput, error) {
	return watch(s, ctx, request, input)
}

func (s *PrepSubsystem) status(ctx context.Context, request *mcp.CallToolRequest, input StatusInput) (*mcp.CallToolResult, StatusOutput, error) {
	return status(s, ctx, request, input)
}

func (s *PrepSubsystem) createEpic(ctx context.Context, request *mcp.CallToolRequest, input EpicInput) (*mcp.CallToolResult, EpicOutput, error) {
	return createEpic(s, ctx, request, input)
}

func (s *PrepSubsystem) createPR(ctx context.Context, request *mcp.CallToolRequest, input CreatePRInput) (*mcp.CallToolResult, CreatePROutput, error) {
	return createPR(s, ctx, request, input)
}

func (s *PrepSubsystem) planCreate(ctx context.Context, request *mcp.CallToolRequest, input PlanCreateInput) (*mcp.CallToolResult, PlanCreateOutput, error) {
	return planCreate(s, ctx, request, input)
}

func (s *PrepSubsystem) createIssue(ctx context.Context, org, repo, title, body string, labelIDs []int64) (ChildRef, error) {
	return createIssue(s, ctx, org, repo, title, body, labelIDs)
}

func (s *PrepSubsystem) cloneWorkspaceDeps(ctx context.Context, workspaceDir, repoDir, org string) error {
	return cloneWorkspaceDeps(s, ctx, workspaceDir, repoDir, org)
}

func (s *PrepSubsystem) copyRepoSpecs(workspaceDir, repo string) error {
	return copyRepoSpecs(s, workspaceDir, repo)
}

func (s *PrepSubsystem) stateStoreErr() error {
	return stateStoreErr(s)
}

func (s *PrepSubsystem) PrepareWorkspace(ctx context.Context, input PrepInput) (*mcp.CallToolResult, PrepOutput, error) {
	return PrepareWorkspace(s, ctx, input)
}

func (s *PrepSubsystem) TestPrepWorkspace(ctx context.Context, input PrepInput) (*mcp.CallToolResult, PrepOutput, error) {
	return TestPrepWorkspace(s, ctx, input)
}

func (s *PrepSubsystem) prepWorkspace(ctx context.Context, request *mcp.CallToolRequest, input PrepInput) (*mcp.CallToolResult, PrepOutput, error) {
	return prepWorkspace(s, ctx, request, input)
}

func (s *PrepSubsystem) runWorkspaceLanguagePrep(ctx context.Context, workspaceDir, repoDir string) error {
	return runWorkspaceLanguagePrep(s, ctx, workspaceDir, repoDir)
}

func (s *PrepSubsystem) templateCreatePlan(ctx context.Context, request *mcp.CallToolRequest, input TemplateCreatePlanInput) (*mcp.CallToolResult, TemplateCreatePlanOutput, error) {
	return templateCreatePlan(s, ctx, request, input)
}

func (s *PrepSubsystem) planFromIssue(ctx context.Context, request *mcp.CallToolRequest, input PlanFromIssueInput) (*mcp.CallToolResult, PlanFromIssueOutput, error) {
	return planFromIssue(s, ctx, request, input)
}

func (s *PrepSubsystem) dispatchStart(ctx context.Context, request *mcp.CallToolRequest, input ShutdownInput) (*mcp.CallToolResult, ShutdownOutput, error) {
	return dispatchStart(s, ctx, request, input)
}

func (s *PrepSubsystem) shutdownGraceful(ctx context.Context, request *mcp.CallToolRequest, input ShutdownInput) (*mcp.CallToolResult, ShutdownOutput, error) {
	return shutdownGraceful(s, ctx, request, input)
}

func (s *PrepSubsystem) shutdownNow(ctx context.Context, request *mcp.CallToolRequest, input ShutdownInput) (*mcp.CallToolResult, ShutdownOutput, error) {
	return shutdownNow(s, ctx, request, input)
}

func (s *PrepSubsystem) pipelineEpicCreate(ctx context.Context, input PipelineEpicCreateInput) (PipelineEpicCreateOutput, error) {
	return pipelineEpicCreate(s, ctx, input)
}

func (s *PrepSubsystem) pipelineEpicRun(ctx context.Context, input PipelineEpicRunInput) (PipelineEpicRunOutput, error) {
	return pipelineEpicRun(s, ctx, input)
}

func (s *PrepSubsystem) pipelineEpicSync(ctx context.Context, org, repo string, number int, dryRun bool) (PipelineEpicSyncOutput, error) {
	return pipelineEpicSync(s, ctx, org, repo, number, dryRun)
}

func (s *PrepSubsystem) pipelineAudit(ctx context.Context, input PipelineAuditInput) (PipelineAuditOutput, error) {
	return pipelineAudit(s, ctx, input)
}

func (s *PrepSubsystem) scan(ctx context.Context, request *mcp.CallToolRequest, input ScanInput) (*mcp.CallToolResult, ScanOutput, error) {
	return scan(s, ctx, request, input)
}

func (s *PrepSubsystem) listOrgRepos(ctx context.Context, org string) ([]string, error) {
	return listOrgRepos(s, ctx, org)
}

func (s *PrepSubsystem) listRepoIssues(ctx context.Context, org, repo, label string) ([]ScanIssue, error) {
	return listRepoIssues(s, ctx, org, repo, label)
}

func (s *PrepSubsystem) pipelineFixReviews(ctx context.Context, input PipelineFixInput) (PipelineFixOutput, error) {
	return pipelineFixReviews(s, ctx, input)
}

func (s *PrepSubsystem) pipelineFixConflicts(ctx context.Context, input PipelineFixInput) (PipelineFixOutput, error) {
	return pipelineFixConflicts(s, ctx, input)
}

func (s *PrepSubsystem) pipelineFixFormat(ctx context.Context, input PipelineFixInput) (PipelineFixOutput, error) {
	return pipelineFixFormat(s, ctx, input)
}

func (s *PrepSubsystem) pipelineFixThreads(ctx context.Context, input PipelineFixInput) (PipelineFixOutput, error) {
	return pipelineFixThreads(s, ctx, input)
}

func (s *PrepSubsystem) pipelineOnboard(ctx context.Context, input PipelineOnboardInput) (PipelineOnboardOutput, error) {
	return pipelineOnboard(s, ctx, input)
}

func (s *PrepSubsystem) pipelineOnboardDispatchDirect(ctx context.Context, input PipelineOnboardInput, issues []PipelineIssueRef) ([]PipelineIssueRef, error) {
	return pipelineOnboardDispatchDirect(s, ctx, input, issues)
}

func (s *PrepSubsystem) pipelineTrainingCapture(ctx context.Context, input PipelineTrainingCaptureInput) (PipelineTrainingCaptureOutput, error) {
	return pipelineTrainingCapture(s, ctx, input)
}

func (s *PrepSubsystem) pipelineTrainingReadDiff(ctx context.Context, org, repo string, number int, meta PipelinePRMeta) (string, string, error) {
	return pipelineTrainingReadDiff(s, ctx, org, repo, number, meta)
}

func (s *PrepSubsystem) pipelineTrainingReadGitDiff(ctx context.Context, org, repo string, meta PipelinePRMeta) (string, string, error) {
	return pipelineTrainingReadGitDiff(s, ctx, org, repo, meta)
}

func (s *PrepSubsystem) pipelineMonitorWithReader(ctx context.Context, input PipelineMonitorInput, reader *MetaReader) (PipelineMonitorOutput, error) {
	return pipelineMonitorWithReader(s, ctx, input, reader)
}

func (s *PrepSubsystem) phaseGet(ctx context.Context, request *mcp.CallToolRequest, input PhaseGetInput) (*mcp.CallToolResult, PhaseOutput, error) {
	return phaseGet(s, ctx, request, input)
}

func (s *PrepSubsystem) phaseUpdateStatus(ctx context.Context, request *mcp.CallToolRequest, input PhaseStatusInput) (*mcp.CallToolResult, PhaseOutput, error) {
	return phaseUpdateStatus(s, ctx, request, input)
}

func (s *PrepSubsystem) phaseAddCheckpoint(ctx context.Context, request *mcp.CallToolRequest, input PhaseCheckpointInput) (*mcp.CallToolResult, PhaseOutput, error) {
	return phaseAddCheckpoint(s, ctx, request, input)
}

func (s *PrepSubsystem) planRead(ctx context.Context, request *mcp.CallToolRequest, input PlanReadInput) (*mcp.CallToolResult, PlanReadOutput, error) {
	return planRead(s, ctx, request, input)
}

func (s *PrepSubsystem) planCreateCompat(ctx context.Context, request *mcp.CallToolRequest, input PlanCreateInput) (*mcp.CallToolResult, PlanCompatibilityCreateOutput, error) {
	return planCreateCompat(s, ctx, request, input)
}

func (s *PrepSubsystem) planGetCompat(ctx context.Context, request *mcp.CallToolRequest, input PlanReadInput) (*mcp.CallToolResult, PlanCompatibilityGetOutput, error) {
	return planGetCompat(s, ctx, request, input)
}

func (s *PrepSubsystem) planUpdateStatusCompat(ctx context.Context, request *mcp.CallToolRequest, input PlanStatusUpdateInput) (*mcp.CallToolResult, PlanCompatibilityGetOutput, error) {
	return planUpdateStatusCompat(s, ctx, request, input)
}

func (s *PrepSubsystem) planArchiveCompat(ctx context.Context, request *mcp.CallToolRequest, input PlanDeleteInput) (*mcp.CallToolResult, PlanArchiveOutput, error) {
	return planArchiveCompat(s, ctx, request, input)
}

func (s *PrepSubsystem) planUpdate(ctx context.Context, request *mcp.CallToolRequest, input PlanUpdateInput) (*mcp.CallToolResult, PlanUpdateOutput, error) {
	return planUpdate(s, ctx, request, input)
}

func (s *PrepSubsystem) planDelete(ctx context.Context, request *mcp.CallToolRequest, input PlanDeleteInput) (*mcp.CallToolResult, PlanDeleteOutput, error) {
	return planDelete(s, ctx, request, input)
}

func (s *PrepSubsystem) planList(ctx context.Context, request *mcp.CallToolRequest, input PlanListInput) (*mcp.CallToolResult, PlanListOutput, error) {
	return planList(s, ctx, request, input)
}

func (s *PrepSubsystem) reviewQueue(ctx context.Context, request *mcp.CallToolRequest, input ReviewQueueInput) (*mcp.CallToolResult, ReviewQueueOutput, error) {
	return reviewQueue(s, ctx, request, input)
}

func (s *PrepSubsystem) forgeCreatePR(ctx context.Context, org, repo, head, base, title, body string) (string, int, error) {
	return forgeCreatePR(s, ctx, org, repo, head, base, title, body)
}

func (s *PrepSubsystem) prGet(ctx context.Context, request *mcp.CallToolRequest, input PRGetInput) (*mcp.CallToolResult, PRGetOutput, error) {
	return prGet(s, ctx, request, input)
}

func (s *PrepSubsystem) prList(ctx context.Context, request *mcp.CallToolRequest, input ListPRsInput) (*mcp.CallToolResult, ListPRsOutput, error) {
	return prList(s, ctx, request, input)
}

func (s *PrepSubsystem) prMerge(ctx context.Context, request *mcp.CallToolRequest, input PRMergeInput) (*mcp.CallToolResult, PRMergeOutput, error) {
	return prMerge(s, ctx, request, input)
}

func (s *PrepSubsystem) closePR(ctx context.Context, request *mcp.CallToolRequest, input ClosePRInput) (*mcp.CallToolResult, ClosePROutput, error) {
	return closePR(s, ctx, request, input)
}

func (s *PrepSubsystem) deleteBranch(ctx context.Context, request *mcp.CallToolRequest, input DeleteBranchInput) (*mcp.CallToolResult, DeleteBranchOutput, error) {
	return deleteBranch(s, ctx, request, input)
}

func (s *PrepSubsystem) promptVersionTool(ctx context.Context, request *mcp.CallToolRequest, input PromptVersionInput) (*mcp.CallToolResult, PromptVersionOutput, error) {
	return promptVersionTool(s, ctx, request, input)
}

func (s *PrepSubsystem) dispatchRemote(ctx context.Context, request *mcp.CallToolRequest, input RemoteDispatchInput) (*mcp.CallToolResult, RemoteDispatchOutput, error) {
	return dispatchRemote(s, ctx, request, input)
}

func (s *PrepSubsystem) listPRs(ctx context.Context, request *mcp.CallToolRequest, input ListPRsInput) (*mcp.CallToolResult, ListPRsOutput, error) {
	return listPRs(s, ctx, request, input)
}

func (s *PrepSubsystem) listRepoPRs(ctx context.Context, org, repo, state string) ([]PRInfo, error) {
	return listRepoPRs(s, ctx, org, repo, state)
}

func (s *PrepSubsystem) templateList(ctx context.Context, request *mcp.CallToolRequest, input TemplateListInput) (*mcp.CallToolResult, TemplateListOutput, error) {
	return templateList(s, ctx, request, input)
}

func (s *PrepSubsystem) templatePreview(ctx context.Context, request *mcp.CallToolRequest, input TemplatePreviewInput) (*mcp.CallToolResult, TemplatePreviewOutput, error) {
	return templatePreview(s, ctx, request, input)
}

func (s *PrepSubsystem) taskCreate(ctx context.Context, request *mcp.CallToolRequest, input TaskCreateInput) (*mcp.CallToolResult, TaskCreateOutput, error) {
	return taskCreate(s, ctx, request, input)
}

func (s *PrepSubsystem) taskUpdate(ctx context.Context, request *mcp.CallToolRequest, input TaskUpdateInput) (*mcp.CallToolResult, TaskOutput, error) {
	return taskUpdate(s, ctx, request, input)
}

func (s *PrepSubsystem) taskToggle(ctx context.Context, request *mcp.CallToolRequest, input TaskToggleInput) (*mcp.CallToolResult, TaskOutput, error) {
	return taskToggle(s, ctx, request, input)
}

func (s *PrepSubsystem) ScheduleRevision(ctx context.Context, pageID, content string) (SEORevision, error) {
	return ScheduleRevision(s, ctx, pageID, content)
}

func (s *PrepSubsystem) GetPendingRevisions(pageID string) ([]SEORevision, error) {
	return GetPendingRevisions(s, pageID)
}

func (s *PrepSubsystem) OnGooglebotVisit(ctx context.Context, pageID string) error {
	return OnGooglebotVisit(s, ctx, pageID)
}

func (s *PrepSubsystem) HandleGooglebotVisit(ctx context.Context, pageID, userAgent string) error {
	return HandleGooglebotVisit(s, ctx, pageID, userAgent)
}

func (s *PrepSubsystem) contentSEOStore() (*store.Store, error) {
	return contentSEOStore(s)
}

func (s *PrepSubsystem) contentSEORevisionRecords(storeInstance *store.Store, pageID string, pendingOnly bool) ([]seoRevisionRecord, error) {
	return contentSEORevisionRecords(s, storeInstance, pageID, pendingOnly)
}

func (s *PrepSubsystem) syncPush(ctx context.Context, agentID string) (SyncPushOutput, error) {
	return syncPush(s, ctx, agentID)
}

func (s *PrepSubsystem) syncPushInput(ctx context.Context, input SyncPushInput) (SyncPushOutput, error) {
	return syncPushInput(s, ctx, input)
}

func (s *PrepSubsystem) syncPull(ctx context.Context, agentID string) (SyncPullOutput, error) {
	return syncPull(s, ctx, agentID)
}

func (s *PrepSubsystem) syncPullInput(ctx context.Context, input SyncPullInput) (SyncPullOutput, error) {
	return syncPullInput(s, ctx, input)
}

type pipelineForgeMetaReader struct {
	subsystem *PrepSubsystem
	org       string
}

func (c RemoteClient) Initialize(ctx context.Context) (string, error) {
	return InitializeRemoteClient(c, ctx)
}

func (c RemoteClient) Call(ctx context.Context, sessionID string, body []byte) ([]byte, error) {
	return CallRemoteClient(c, ctx, sessionID, body)
}

func (r *pipelineForgeMetaReader) GetPRMeta(ctx context.Context, repo string, prNumber int) (PipelinePRMeta, error) {
	return newPipelineForgeMetaReader(r.subsystem, r.org).GetPRMeta(ctx, repo, prNumber)
}

func (r *pipelineForgeMetaReader) GetEpicMeta(ctx context.Context, repo string, issueNumber int) (PipelineEpicMeta, error) {
	return newPipelineForgeMetaReader(r.subsystem, r.org).GetEpicMeta(ctx, repo, issueNumber)
}

func (r *pipelineForgeMetaReader) GetIssueState(ctx context.Context, repo string, issueNumber int) (PipelineIssueState, error) {
	return newPipelineForgeMetaReader(r.subsystem, r.org).GetIssueState(ctx, repo, issueNumber)
}

func (r *pipelineForgeMetaReader) GetCommentReactions(ctx context.Context, repo string, commentID int64) ([]PipelineReactionMeta, error) {
	return newPipelineForgeMetaReader(r.subsystem, r.org).GetCommentReactions(ctx, repo, commentID)
}

// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package repo

import (
	"fmt"

	"github.com/gogits/gogs/models"
	"github.com/gogits/gogs/modules/auth"
	"github.com/gogits/gogs/modules/base"
	"github.com/gogits/gogs/modules/middleware"
)

const (
	PULLS    base.TplName = "repo/pulls"
	NEW_PULL base.TplName = "repo/pull_new"
)

func Pulls(ctx *middleware.Context) {
	ctx.Data["IsRepoToolbarPulls"] = true
	ctx.HTML(200, PULLS)
}

func hasPullRequested(ctx *middleware.Context, repoID int64, forkRepo *models.Repository) bool {
	pr, err := models.GetPullRequest(repoID)
	if err != nil {
		if err != models.ErrPullRequestNotExist {
			ctx.Handle(500, "GetPullRequest", err)
			return true
		}
	} else {
		repoLink, err := forkRepo.RepoLink()
		if err != nil {
			ctx.Handle(500, "RepoLink", err)
		} else {
			ctx.Redirect(fmt.Sprintf("%s/pulls/%d", repoLink, pr.Index))
		}
		return true
	}
	return false
}

func NewPullRequest(ctx *middleware.Context) {
	repo := ctx.Repo.Repository
	if !repo.IsFork {
		ctx.Redirect(ctx.Repo.RepoLink)
		return
	}
	ctx.Data["RequestFrom"] = repo.Owner.Name + "/" + repo.Name

	if err := repo.GetForkRepo(); err != nil {
		ctx.Handle(500, "GetForkRepo", err)
		return
	}
	forkRepo := repo.ForkRepo

	if hasPullRequested(ctx, repo.ID, forkRepo) {
		return
	}

	if err := forkRepo.GetBranches(); err != nil {
		ctx.Handle(500, "GetBranches", err)
		return
	}
	ctx.Data["ForkRepo"] = forkRepo
	ctx.Data["RequestTo"] = forkRepo.Owner.Name + "/" + forkRepo.Name

	if len(forkRepo.DefaultBranch) == 0 {
		forkRepo.DefaultBranch = forkRepo.Branches[0]
	}
	ctx.Data["DefaultBranch"] = forkRepo.DefaultBranch

	ctx.HTML(200, NEW_PULL)
}

// FIXME: check if branch exists
func NewPullRequestPost(ctx *middleware.Context, form auth.NewPullRequestForm) {
	repo := ctx.Repo.Repository
	if err := repo.GetForkRepo(); err != nil {
		ctx.Handle(500, "GetForkRepo", err)
		return
	}
	forkRepo := repo.ForkRepo

	if hasPullRequested(ctx, repo.ID, forkRepo) {
		return
	}

	pr := &models.Issue{
		RepoID:   repo.ForkID,
		Index:    int64(forkRepo.NumIssues) + 1,
		Name:     form.Title,
		PosterID: ctx.User.Id,
		IsPull:   true,
		Content:  form.Description,
	}
	pullRepo := &models.PullRepo{
		FromRepoID: repo.ID,
		ToRepoID:   forkRepo.ID,
		FromBranch: form.FromBranch,
		ToBranch:   form.ToBranch,
	}
	if err := models.NewPullRequest(pr, pullRepo); err != nil {
		ctx.Handle(500, "NewPullRequest", err)
		return
	} else if err := models.NewIssueUserPairs(forkRepo, pr.ID, forkRepo.OwnerID,
		ctx.User.Id, 0); err != nil {
		ctx.Handle(500, "NewIssueUserPairs", err)
		return
	}

	// FIXME: add action

	repoLink, err := forkRepo.RepoLink()
	if err != nil {
		ctx.Handle(500, "RepoLink", err)
		return
	}
	ctx.Redirect(fmt.Sprintf("%s/pulls/%d", repoLink, pr.Index))
}

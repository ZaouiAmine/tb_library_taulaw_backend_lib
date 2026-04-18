package lib

import httpevent "github.com/taubyte/go-sdk/http/event"

func routeDeleteAssistants(h httpevent.Event) uint32 {
	return grDelete(h, "assistants")
}

func routeDeleteBannersById(h httpevent.Event) uint32 {
	return handleDeleteBanner(h)
}

func routeDeleteCaseCategoriesDeleteById(h httpevent.Event) uint32 {
	return localizedDelete(h, caseCategoriesMatcher, "case_categories/", "delete")
}

func routeDeleteCaseChambersDeleteById(h httpevent.Event) uint32 {
	return localizedDelete(h, caseChambersMatcher, "case_chambers/", "delete")
}

func routeDeleteCasePhasesDeleteById(h httpevent.Event) uint32 {
	return localizedDelete(h, casePhasesMatcher, "case_phases/", "delete")
}

func routeDeleteCasesByIdDelete(h httpevent.Event) uint32 {
	return grDelete(h, "cases")
}

func routeDeleteCasesByIdFiles(h httpevent.Event) uint32 {
	return grDelete(h, "cases")
}

func routeDeleteConsultationPackagesById(h httpevent.Event) uint32 {
	return handleDeleteConsultationPackage(h)
}

func routeDeleteFeedsClientCasesById(h httpevent.Event) uint32 {
	return grDelete(h, "feeds_client_cases")
}

func routeDeleteFeedsClientConsultationsById(h httpevent.Event) uint32 {
	return grDelete(h, "feeds_client_consultations")
}

func routeDeleteFeedsClientConsultationsByIdAnswersByAnswerId(h httpevent.Event) uint32 {
	return grDelete(h, "feeds_client_consultations")
}

func routeDeleteFeedsLawyerConsultationsAnswersById(h httpevent.Event) uint32 {
	return grDelete(h, "feeds_lawyer_consultations")
}

func routeDeleteFeedsLawyerConsultationsCommentsByCommentId(h httpevent.Event) uint32 {
	return grDelete(h, "feeds_lawyer_consultations")
}

func routeDeleteFoldersById(h httpevent.Event) uint32 {
	return grDelete(h, "folders")
}

func routeDeleteFoldersByIdShare(h httpevent.Event) uint32 {
	return grDelete(h, "folders")
}

func routeDeleteFoldersByIdShareAll(h httpevent.Event) uint32 {
	return grDelete(h, "folders")
}

func routeDeleteFoldersFilesById(h httpevent.Event) uint32 {
	return grDelete(h, "folders")
}

func routeDeleteFoldersFilesByIdShare(h httpevent.Event) uint32 {
	return grDelete(h, "folders")
}

func routeDeleteFoldersFilesByIdShareAll(h httpevent.Event) uint32 {
	return grDelete(h, "folders")
}

func routeDeleteForumPostsById(h httpevent.Event) uint32 {
	return grDelete(h, "forum_posts")
}

func routeDeleteForumRepliesById(h httpevent.Event) uint32 {
	return grDelete(h, "forum_replies")
}

func routeDeleteLawyerPackagesById(h httpevent.Event) uint32 {
	return handleDeleteLawyerPackage(h)
}

func routeDeletePetitionsById(h httpevent.Event) uint32 {
	return grDelete(h, "petitions")
}

func routeDeleteRequestsById(h httpevent.Event) uint32 {
	return grDelete(h, "requests")
}

func routeDeleteSpecializationsDeleteById(h httpevent.Event) uint32 {
	return localizedDelete(h, specializationsMatcher, "specializations/", "delete")
}

func routeDeleteStatesDeleteById(h httpevent.Event) uint32 {
	return handleDeleteStatesDeleteById(h)
}

func routeDeleteUserConsultationSubscriptionsById(h httpevent.Event) uint32 {
	return grDelete(h, "user_consultation_subscriptions")
}

func routeGetAdminReports(h httpevent.Event) uint32 {
	return handleGetAdminReportsNest(h)
}

func routeGetAssistants(h httpevent.Event) uint32 {
	return grList(h, "assistants")
}

func routeGetAssistantsByIdPermissions(h httpevent.Event) uint32 {
	return grList(h, "assistants")
}

func routeGetBanners(h httpevent.Event) uint32 {
	return handleGetBannersPublic(h, false)
}

func routeGetBannersActiveByType(h httpevent.Event) uint32 {
	return handleGetBannersPublic(h, true)
}

func routeGetBannersById(h httpevent.Event) uint32 {
	return handleBannerByID(h)
}

func routeGetCaseCategoriesGet(h httpevent.Event) uint32 {
	return localizedGetPublic(h, caseCategoriesMatcher, "case_categories/")
}

func routeGetCaseCategoriesIndex(h httpevent.Event) uint32 {
	return localizedIndex(h, caseCategoriesMatcher, "case_categories/", "")
}

func routeGetCaseChambersGet(h httpevent.Event) uint32 {
	return localizedGetPublic(h, caseChambersMatcher, "case_chambers/")
}

func routeGetCaseChambersIndex(h httpevent.Event) uint32 {
	return localizedIndex(h, caseChambersMatcher, "case_chambers/", "")
}

func routeGetCasePhasesGet(h httpevent.Event) uint32 {
	return localizedGetPublic(h, casePhasesMatcher, "case_phases/")
}

func routeGetCasePhasesIndex(h httpevent.Event) uint32 {
	return localizedIndex(h, casePhasesMatcher, "case_phases/", "")
}

func routeGetCasesByIdDownload(h httpevent.Event) uint32 {
	return grOKMap(h, map[string]any{"url": "/cases/download"})
}

func routeGetCasesByIdGetCaseFiles(h httpevent.Event) uint32 {
	return grGetByID(h, "cases")
}

func routeGetCasesByIdGetCaseNotes(h httpevent.Event) uint32 {
	return grGetByID(h, "cases")
}

func routeGetCasesByIdUsersPermissions(h httpevent.Event) uint32 {
	return grGetByID(h, "cases")
}

func routeGetCasesGet(h httpevent.Event) uint32 {
	return grList(h, "cases")
}

func routeGetConsultationPackages(h httpevent.Event) uint32 {
	return handleListConsultationPackagesNest(h)
}

func routeGetConsultationPackagesById(h httpevent.Event) uint32 {
	return handleConsultationPackageByID(h)
}

func routeGetFeedsClientCases(h httpevent.Event) uint32 {
	return grList(h, "feeds_client_cases")
}

func routeGetFeedsClientCasesApplicationsAccepted(h httpevent.Event) uint32 {
	return grList(h, "feeds_client_cases")
}

func routeGetFeedsClientCasesById(h httpevent.Event) uint32 {
	return grList(h, "feeds_client_cases")
}

func routeGetFeedsClientCasesByIdApplications(h httpevent.Event) uint32 {
	return grList(h, "feeds_client_cases")
}

func routeGetFeedsClientConsultations(h httpevent.Event) uint32 {
	return grList(h, "feeds_client_consultations")
}

func routeGetFeedsClientConsultationsAnswers(h httpevent.Event) uint32 {
	return grList(h, "feeds_client_consultations")
}

func routeGetFeedsClientConsultationsByIdAnswers(h httpevent.Event) uint32 {
	return grList(h, "feeds_client_consultations")
}

func routeGetFeedsLawyerCases(h httpevent.Event) uint32 {
	return grList(h, "feeds_lawyer_cases")
}

func routeGetFeedsLawyerCasesLawyerCases(h httpevent.Event) uint32 {
	return grList(h, "feeds_lawyer_cases")
}

func routeGetFeedsLawyerConsultations(h httpevent.Event) uint32 {
	return grList(h, "feeds_lawyer_consultations")
}

func routeGetFeedsLawyerConsultationsAnswersMe(h httpevent.Event) uint32 {
	return grList(h, "feeds_lawyer_consultations")
}

func routeGetFeedsLawyerConsultationsById(h httpevent.Event) uint32 {
	return grList(h, "feeds_lawyer_consultations")
}

func routeGetFeedsLawyerConsultationsByIdAnswers(h httpevent.Event) uint32 {
	return grList(h, "feeds_lawyer_consultations")
}

func routeGetFeedsLawyerConsultationsCommentsByAnswerId(h httpevent.Event) uint32 {
	return grList(h, "feeds_lawyer_consultations")
}

func routeGetFolders(h httpevent.Event) uint32 {
	return grList(h, "folders")
}

func routeGetFoldersById(h httpevent.Event) uint32 {
	return grList(h, "folders")
}

func routeGetFoldersByIdDownloadFolder(h httpevent.Event) uint32 {
	return grOKMap(h, map[string]any{"url": "/folders/dl"})
}

func routeGetFoldersByIdShare(h httpevent.Event) uint32 {
	return grList(h, "folders")
}

func routeGetFoldersFilesByIdDownload(h httpevent.Event) uint32 {
	return grOKMap(h, map[string]any{"url": "/folders/file/dl"})
}

func routeGetFoldersFilesByIdShare(h httpevent.Event) uint32 {
	return grList(h, "folder_file_shares")
}

func routeGetFoldersSharesReceived(h httpevent.Event) uint32 {
	return grList(h, "folder_shares")
}

func routeGetFoldersSharesReceivedUnreadCount(h httpevent.Event) uint32 {
	return grOKMap(h, map[string]any{"count": 0})
}

func routeGetFoldersSharesSent(h httpevent.Event) uint32 {
	return grList(h, "folder_shares")
}

func routeGetForumNotifications(h httpevent.Event) uint32 {
	return grList(h, "forum_notifications")
}

func routeGetForumPosts(h httpevent.Event) uint32 {
	return grList(h, "forum_posts")
}

func routeGetForumPostsById(h httpevent.Event) uint32 {
	return grGetByID(h, "forum_posts")
}

func routeGetForumPostsByIdReplies(h httpevent.Event) uint32 {
	return grList(h, "forum_replies")
}

func routeGetJudicalReqiests(h httpevent.Event) uint32 {
	return grList(h, "judicial_requests")
}

func routeGetJudicalReqiestsById(h httpevent.Event) uint32 {
	return grGetByID(h, "judicial_requests")
}

func routeGetLawyerPackages(h httpevent.Event) uint32 {
	return handleListLawyerPackagesNest(h)
}

func routeGetLawyerPackagesById(h httpevent.Event) uint32 {
	return handleLawyerPackageByID(h)
}

func routeGetLawyerPackagesSubscriptionsHistory(h httpevent.Event) uint32 {
	return grOKMap(h, map[string]any{"status": "ok"})
}

func routeGetLawyerSubscriptionsAvailablePackages(h httpevent.Event) uint32 {
	return grOKMap(h, map[string]any{"status": "ok"})
}

func routeGetLawyerSubscriptionsHistory(h httpevent.Event) uint32 {
	return grOKMap(h, map[string]any{"status": "ok"})
}

func routeGetLawyerSubscriptionsStatus(h httpevent.Event) uint32 {
	return grOKMap(h, map[string]any{"status": "ok"})
}

func routeGetPermissions(h httpevent.Event) uint32 {
	return grList(h, "assistants")
}

func routeGetPetitionsById(h httpevent.Event) uint32 {
	return grGetByID(h, "petitions")
}

func routeGetPetitionsByIdPdf(h httpevent.Event) uint32 {
	return grOKMap(h, map[string]any{"url": "/petitions/pdf/" + pathLast(h)})
}

func routeGetPetitionsFinal(h httpevent.Event) uint32 {
	return grList(h, "petitions")
}

func routeGetRequests(h httpevent.Event) uint32 {
	return grList(h, "requests")
}

func routeGetRequestsMyRequests(h httpevent.Event) uint32 {
	return grList(h, "requests")
}

func routeGetRequestsOfficerAccepted(h httpevent.Event) uint32 {
	return grList(h, "requests")
}

func routeGetSpecializationsGet(h httpevent.Event) uint32 {
	return localizedGetPublic(h, specializationsMatcher, "specializations/")
}

func routeGetSpecializationsIndex(h httpevent.Event) uint32 {
	return localizedIndex(h, specializationsMatcher, "specializations/", "")
}

func routeGetStatesGet(h httpevent.Event) uint32 {
	return handleGetStatesGet(h)
}

func routeGetStatesIndex(h httpevent.Event) uint32 {
	return handleGetStatesIndex(h)
}

func routeGetUserConsultationSubscriptions(h httpevent.Event) uint32 {
	return grList(h, "user_consultation_subscriptions")
}

func routeGetUserConsultationSubscriptionsById(h httpevent.Event) uint32 {
	return grGetByID(h, "user_consultation_subscriptions")
}

func routeGetUsersJudicialOfficers(h httpevent.Event) uint32 {
	return handleGetUsersJudicialOfficers(h)
}

func routeGetUsersJudicialOfficersById(h httpevent.Event) uint32 {
	return handleGetUsersJudicialOfficersById(h)
}

func routeGetUsersLawyers(h httpevent.Event) uint32 {
	return handleGetUsersLawyers(h)
}

func routeGetUsersLawyersById(h httpevent.Event) uint32 {
	return handleGetUsersLawyersById(h)
}

func routeGetUsersLawyersDashboardStats(h httpevent.Event) uint32 {
	return handleGetUsersLawyersDashboardStats(h)
}

func routeGetUsersLawyersGet(h httpevent.Event) uint32 {
	return handleGetUsersLawyersGet(h)
}

func routeGetUsersLawyersLawyers(h httpevent.Event) uint32 {
	return handleGetUsersLawyersLawyers(h)
}

func routeGetUsersLawyersLawyersVerifiying(h httpevent.Event) uint32 {
	return handleGetUsersLawyersLawyersVerifiying(h)
}

func routeGetUsersMe(h httpevent.Event) uint32 {
	return handleGetUsersMe(h)
}

func routeGetUsersSearch(h httpevent.Event) uint32 {
	return handleGetUsersSearch(h)
}

func routePatchAssistantsByIdPermissions(h httpevent.Event) uint32 {
	return grPatchJSON(h, "assistants")
}

func routePatchBannersById(h httpevent.Event) uint32 {
	return handlePatchBanner(h)
}

func routePatchCaseCategoriesUpdateById(h httpevent.Event) uint32 {
	return localizedPatch(h, caseCategoriesMatcher, "case_categories/", "update")
}

func routePatchCaseChambersUpdateById(h httpevent.Event) uint32 {
	return localizedPatch(h, caseChambersMatcher, "case_chambers/", "update")
}

func routePatchCasePhasesUpdateById(h httpevent.Event) uint32 {
	return localizedPatch(h, casePhasesMatcher, "case_phases/", "update")
}

func routePatchCasesByIdUpdate(h httpevent.Event) uint32 {
	return grPatchJSON(h, "cases")
}

func routePatchConsultationPackagesById(h httpevent.Event) uint32 {
	return handlePatchConsultationPackage(h)
}

func routePatchFeedsClientCasesById(h httpevent.Event) uint32 {
	return grPatchJSON(h, "feeds_client_cases")
}

func routePatchFeedsClientConsultationsById(h httpevent.Event) uint32 {
	return grPatchJSON(h, "feeds_client_consultations")
}

func routePatchFeedsLawyerConsultationsAnswersById(h httpevent.Event) uint32 {
	return grPatchJSON(h, "feeds_lawyer_consultations")
}

func routePatchFeedsLawyerConsultationsCommentsByCommentId(h httpevent.Event) uint32 {
	return grPatchJSON(h, "feeds_lawyer_consultations")
}

func routePatchFoldersById(h httpevent.Event) uint32 {
	return grPatchJSON(h, "folders")
}

func routePatchFoldersFilesByIdRename(h httpevent.Event) uint32 {
	return grPatchJSON(h, "folders")
}

func routePatchForumPostsById(h httpevent.Event) uint32 {
	return grPatchJSON(h, "forum_posts")
}

func routePatchLawyerPackagesById(h httpevent.Event) uint32 {
	return handlePatchLawyerPackage(h)
}

func routePatchRequestsById(h httpevent.Event) uint32 {
	return grPatchJSON(h, "requests")
}

func routePatchSpecializationsUpdateById(h httpevent.Event) uint32 {
	return localizedPatch(h, specializationsMatcher, "specializations/", "update")
}

func routePatchStatesUpdateById(h httpevent.Event) uint32 {
	return handlePatchStatesUpdateById(h)
}

func routePatchUserConsultationSubscriptionsById(h httpevent.Event) uint32 {
	return grPatchJSON(h, "user_consultation_subscriptions")
}

func routePatchUsersChangePassword(h httpevent.Event) uint32 {
	return handlePatchUsersChangePassword(h)
}

func routePatchUsersLawyersAcceptVerifiyingById(h httpevent.Event) uint32 {
	return handlePatchUsersLawyersAcceptVerifiyingById(h)
}

func routePatchUsersMe(h httpevent.Event) uint32 {
	return handlePatchUsersMe(h)
}

func routePostAdminReportsAction(h httpevent.Event) uint32 {
	return handlePostAdminReportsActionNest(h)
}

func routePostAssistants(h httpevent.Event) uint32 {
	return grPostJSON(h, "assistants")
}

func routePostAuthLogin(h httpevent.Event) uint32 {
	return handlePostAuthLogin(h)
}

func routePostAuthLoginGoogle(h httpevent.Event) uint32 {
	return handlePostAuthLoginGoogle(h)
}

func routePostAuthLogout(h httpevent.Event) uint32 {
	return handlePostAuthLogout(h)
}

func routePostAuthRefreshToken(h httpevent.Event) uint32 {
	return handlePostAuthRefreshToken(h)
}

func routePostAuthRegister(h httpevent.Event) uint32 {
	return handlePostAuthRegister(h)
}

func routePostAuthResendResetCode(h httpevent.Event) uint32 {
	return handlePostAuthResendResetCode(h)
}

func routePostAuthResendVerificationCode(h httpevent.Event) uint32 {
	return handlePostAuthResendVerificationCode(h)
}

func routePostAuthResetPassword(h httpevent.Event) uint32 {
	return handlePostAuthResetPassword(h)
}

func routePostAuthSetRole(h httpevent.Event) uint32 {
	return handlePostAuthSetRole(h)
}

func routePostAuthVerifyEmail(h httpevent.Event) uint32 {
	return handlePostAuthVerifyEmail(h)
}

func routePostAuthVerifyResetCode(h httpevent.Event) uint32 {
	return handlePostAuthVerifyResetCode(h)
}

func routePostBanners(h httpevent.Event) uint32 {
	return handlePostBanner(h)
}

func routePostBotChat(h httpevent.Event) uint32 {
	return grOKMap(h, map[string]any{"reply": "ok"})
}

func routePostCaseCategoriesStore(h httpevent.Event) uint32 {
	return localizedStore(h, caseCategoriesMatcher, "case_categories/", "category")
}

func routePostCaseChambersStore(h httpevent.Event) uint32 {
	return localizedStore(h, caseChambersMatcher, "case_chambers/", "chamber")
}

func routePostCasePhasesStore(h httpevent.Event) uint32 {
	return localizedStore(h, casePhasesMatcher, "case_phases/", "phase")
}

func routePostCasesByIdFileTitle(h httpevent.Event) uint32 {
	return grPostJSON(h, "cases")
}

func routePostCasesByIdFiles(h httpevent.Event) uint32 {
	return grPostJSON(h, "cases")
}

func routePostCasesByIdNotes(h httpevent.Event) uint32 {
	return grPostJSON(h, "cases")
}

func routePostCasesByIdSaveCase(h httpevent.Event) uint32 {
	return grPostJSON(h, "cases")
}

func routePostCasesByIdShare(h httpevent.Event) uint32 {
	return grPostJSON(h, "cases")
}

func routePostCasesStore(h httpevent.Event) uint32 {
	return grPostJSON(h, "cases")
}

func routePostConsultationPackages(h httpevent.Event) uint32 {
	return handlePostConsultationPackage(h)
}

func routePostFeedsClientCases(h httpevent.Event) uint32 {
	return grPostJSON(h, "feeds_client_cases")
}

func routePostFeedsClientCasesByIdApplicationsAcceptByApplicationId(h httpevent.Event) uint32 {
	return grPostJSON(h, "feeds_client_cases")
}

func routePostFeedsClientCasesByIdApplicationsRejectByApplicationId(h httpevent.Event) uint32 {
	return grPostJSON(h, "feeds_client_cases")
}

func routePostFeedsClientConsultations(h httpevent.Event) uint32 {
	return grPostJSON(h, "feeds_client_consultations")
}

func routePostFeedsClientConsultationsByIdAnswersByAnswerId(h httpevent.Event) uint32 {
	return grPostJSON(h, "feeds_client_consultations")
}

func routePostFeedsLawyerCasesByIdApply(h httpevent.Event) uint32 {
	return grPostJSON(h, "feeds_lawyer_cases")
}

func routePostFeedsLawyerConsultationsByIdAnswers(h httpevent.Event) uint32 {
	return grPostJSON(h, "feeds_lawyer_consultations")
}

func routePostFeedsLawyerConsultationsCommentsByAnswerId(h httpevent.Event) uint32 {
	return grPostJSON(h, "feeds_lawyer_consultations")
}

func routePostFolders(h httpevent.Event) uint32 {
	return grPostJSON(h, "folders")
}

func routePostFoldersByIdFiles(h httpevent.Event) uint32 {
	return grPostJSON(h, "folders")
}

func routePostFoldersByIdShare(h httpevent.Event) uint32 {
	return grPostJSON(h, "folders")
}

func routePostFoldersFilesByIdShare(h httpevent.Event) uint32 {
	return grPostJSON(h, "folders")
}

func routePostForumPosts(h httpevent.Event) uint32 {
	return grPostJSON(h, "forum_posts")
}

func routePostForumPostsHide(h httpevent.Event) uint32 {
	return grPatchJSON(h, "forum_posts")
}

func routePostForumPostsReport(h httpevent.Event) uint32 {
	return grPostJSON(h, "forum_posts")
}

func routePostForumReplies(h httpevent.Event) uint32 {
	return grPostJSON(h, "forum_replies")
}

func routePostForumRepliesReport(h httpevent.Event) uint32 {
	return grPostJSON(h, "forum_replies")
}

func routePostLawyerPackages(h httpevent.Event) uint32 {
	return handlePostLawyerPackage(h)
}

func routePostLawyerPackagesSubscriptionsCleanup(h httpevent.Event) uint32 {
	return grOKMap(h, map[string]any{"status": "ok"})
}

func routePostLawyerSubscriptionsSubscribe(h httpevent.Event) uint32 {
	return grOKMap(h, map[string]any{"status": "ok"})
}

func routePostPetitions(h httpevent.Event) uint32 {
	return grPostJSON(h, "petitions")
}

func routePostPetitionsByIdMove(h httpevent.Event) uint32 {
	return grPatchJSON(h, "petitions")
}

func routePostPetitionsImage(h httpevent.Event) uint32 {
	return grPostJSON(h, "petitions")
}

func routePostPetitionsUploadFile(h httpevent.Event) uint32 {
	return grPostJSON(h, "petitions")
}

func routePostRequests(h httpevent.Event) uint32 {
	return grPostJSON(h, "requests")
}

func routePostRequestsByIdUpdateStatus(h httpevent.Event) uint32 {
	return grPatchJSON(h, "requests")
}

func routePostSpecializationsStore(h httpevent.Event) uint32 {
	return localizedStore(h, specializationsMatcher, "specializations/", "spec")
}

func routePostStatesStore(h httpevent.Event) uint32 {
	return handlePostStatesStore(h)
}

func routePostUserConsultationSubscriptions(h httpevent.Event) uint32 {
	return grPostJSON(h, "user_consultation_subscriptions")
}

func routePostUsersChatNotification(h httpevent.Event) uint32 {
	return grOKMap(h, map[string]any{"sent": true})
}

func routePostUsersCreateImage(h httpevent.Event) uint32 {
	return handlePostUsersCreateImage(h)
}

func routePostUsersLawyerVerificationFiles(h httpevent.Event) uint32 {
	return grOKMap(h, map[string]any{"uploaded": true})
}

func routePostUsersRemoveImage(h httpevent.Event) uint32 {
	return handlePostUsersRemoveImage(h)
}

func routePostUsersRemoveImageCover(h httpevent.Event) uint32 {
	return handlePostUsersRemoveImageCover(h)
}

func routePostUsersSavedAiImage(h httpevent.Event) uint32 {
	return handlePostUsersSavedAiImage(h)
}

func routePostUsersUploadImage(h httpevent.Event) uint32 {
	return handlePostUsersUploadImage(h)
}

func routePostUsersUploadImageCover(h httpevent.Event) uint32 {
	return handlePostUsersUploadImageCover(h)
}

func routePutCasesByIdRelations(h httpevent.Event) uint32 {
	return grPatchJSON(h, "cases")
}

func routePutForumNotificationsByIdRead(h httpevent.Event) uint32 {
	return grPatchJSON(h, "forum_notifications")
}

func routePutForumRepliesById(h httpevent.Event) uint32 {
	return grPatchJSON(h, "forum_replies")
}

func routePutPetitionsById(h httpevent.Event) uint32 {
	return grPatchJSON(h, "petitions")
}

func routePutUsersLawyersAcceptByUserId(h httpevent.Event) uint32 {
	return handlePutUsersLawyersAcceptByUserId(h)
}

func routePutUsersLawyersRejectByUserId(h httpevent.Event) uint32 {
	return handlePutUsersLawyersRejectByUserId(h)
}

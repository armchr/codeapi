<%@ page contentType="text/html;charset=UTF-8" %>
<%@ taglib prefix="c" uri="jakarta.tags.core" %>
<jsp:include page="../includes/header.jsp">
    <jsp:param name="title" value="${pet.name}"/>
</jsp:include>

<nav aria-label="breadcrumb" class="mb-4">
    <ol class="breadcrumb">
        <li class="breadcrumb-item"><a href="${pageContext.request.contextPath}/home">Home</a></li>
        <li class="breadcrumb-item"><a href="${pageContext.request.contextPath}/owners">Owners</a></li>
        <li class="breadcrumb-item"><a href="${pageContext.request.contextPath}/owners/${owner.id}">${owner.fullName}</a></li>
        <li class="breadcrumb-item active">${pet.name}</li>
    </ol>
</nav>

<div class="row">
    <div class="col-md-6">
        <div class="card">
            <div class="card-header d-flex justify-content-between align-items-center">
                <h4 class="mb-0"><i class="bi bi-heart-fill text-danger me-2"></i>${pet.name}</h4>
                <a href="${pageContext.request.contextPath}/owners/${owner.id}/pets/${pet.id}/edit"
                   class="btn btn-sm btn-outline-primary">
                    <i class="bi bi-pencil me-1"></i>Edit
                </a>
            </div>
            <div class="card-body">
                <table class="table table-borderless">
                    <tr>
                        <th width="30%">Type</th>
                        <td><span class="badge bg-primary">${pet.typeName}</span></td>
                    </tr>
                    <tr>
                        <th>Birth Date</th>
                        <td>${pet.birthDate}</td>
                    </tr>
                    <tr>
                        <th>Age</th>
                        <td>${pet.age} years</td>
                    </tr>
                    <tr>
                        <th>Owner</th>
                        <td>
                            <a href="${pageContext.request.contextPath}/owners/${owner.id}">
                                ${owner.fullName}
                            </a>
                        </td>
                    </tr>
                </table>
            </div>
        </div>
    </div>

    <div class="col-md-6">
        <div class="card">
            <div class="card-header d-flex justify-content-between align-items-center">
                <h5 class="mb-0"><i class="bi bi-calendar-check me-2"></i>Visits</h5>
                <a href="${pageContext.request.contextPath}/owners/${owner.id}/pets/${pet.id}/visits/new"
                   class="btn btn-sm btn-success">
                    <i class="bi bi-plus-lg me-1"></i>Add Visit
                </a>
            </div>
            <div class="card-body">
                <a href="${pageContext.request.contextPath}/owners/${owner.id}/pets/${pet.id}/visits"
                   class="btn btn-outline-info">
                    <i class="bi bi-calendar-check me-1"></i>View All Visits
                </a>
            </div>
        </div>
    </div>
</div>

<jsp:include page="../includes/footer.jsp"/>

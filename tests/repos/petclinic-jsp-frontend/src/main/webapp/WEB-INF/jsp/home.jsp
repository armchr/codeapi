<%@ page contentType="text/html;charset=UTF-8" %>
<%@ taglib prefix="c" uri="jakarta.tags.core" %>
<jsp:include page="includes/header.jsp">
    <jsp:param name="title" value="Home"/>
</jsp:include>

<div class="hero-section text-center rounded">
    <div class="container">
        <h1><i class="bi bi-heart-pulse-fill me-2"></i>Welcome to PetClinic</h1>
        <p class="lead">A comprehensive pet clinic management system</p>
    </div>
</div>

<div class="row g-4">
    <div class="col-md-6">
        <div class="card h-100">
            <div class="card-body">
                <h5 class="card-title">
                    <i class="bi bi-people text-primary me-2"></i>Pet Owners
                </h5>
                <p class="card-text">
                    Manage pet owners, their contact information, and associated pets.
                    Search for owners by last name or browse all owners.
                </p>
                <a href="${pageContext.request.contextPath}/owners" class="btn btn-primary">
                    <i class="bi bi-arrow-right me-1"></i>View Owners
                </a>
                <a href="${pageContext.request.contextPath}/owners/new" class="btn btn-outline-primary">
                    <i class="bi bi-plus-lg me-1"></i>Add Owner
                </a>
            </div>
        </div>
    </div>

    <div class="col-md-6">
        <div class="card h-100">
            <div class="card-body">
                <h5 class="card-title">
                    <i class="bi bi-person-badge text-success me-2"></i>Veterinarians
                </h5>
                <p class="card-text">
                    View our team of veterinarians and their specialties.
                    Find the right vet for your pet's needs.
                </p>
                <a href="${pageContext.request.contextPath}/vets" class="btn btn-success">
                    <i class="bi bi-arrow-right me-1"></i>View Vets
                </a>
            </div>
        </div>
    </div>
</div>

<div class="row mt-4">
    <div class="col-12">
        <div class="card">
            <div class="card-header">
                <h5 class="mb-0"><i class="bi bi-search me-2"></i>Find Owner</h5>
            </div>
            <div class="card-body">
                <form action="${pageContext.request.contextPath}/owners" method="get" class="row g-3">
                    <div class="col-md-8">
                        <input type="text" class="form-control" name="lastName"
                               placeholder="Enter owner's last name...">
                    </div>
                    <div class="col-md-4">
                        <button type="submit" class="btn btn-primary w-100">
                            <i class="bi bi-search me-1"></i>Search
                        </button>
                    </div>
                </form>
            </div>
        </div>
    </div>
</div>

<div class="row mt-4 g-4">
    <div class="col-md-4">
        <div class="card stats-card h-100">
            <div class="card-body text-center">
                <i class="bi bi-people display-6"></i>
                <h6 class="mt-2">Owners</h6>
                <p class="text-muted small">Manage pet owner records</p>
            </div>
        </div>
    </div>
    <div class="col-md-4">
        <div class="card stats-card h-100">
            <div class="card-body text-center">
                <i class="bi bi-github display-6"></i>
                <h6 class="mt-2">Pets</h6>
                <p class="text-muted small">Track pets and their health</p>
            </div>
        </div>
    </div>
    <div class="col-md-4">
        <div class="card stats-card h-100">
            <div class="card-body text-center">
                <i class="bi bi-calendar-check display-6"></i>
                <h6 class="mt-2">Visits</h6>
                <p class="text-muted small">Schedule and record visits</p>
            </div>
        </div>
    </div>
</div>

<jsp:include page="includes/footer.jsp"/>

package com.example.petclinic.service;

import com.example.petclinic.model.*;
import com.google.gson.*;
import com.google.gson.reflect.TypeToken;

import java.io.IOException;
import java.lang.reflect.Type;
import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.time.Duration;
import java.time.LocalDate;
import java.time.format.DateTimeFormatter;
import java.util.List;

/**
 * HTTP client for communicating with the Spring PetClinic REST API.
 */
public class PetClinicApiClient {
    private static final String DEFAULT_BASE_URL = "http://localhost:9966/petclinic/api";

    private final String baseUrl;
    private final HttpClient httpClient;
    private final Gson gson;

    public PetClinicApiClient() {
        this(DEFAULT_BASE_URL);
    }

    public PetClinicApiClient(String baseUrl) {
        this.baseUrl = baseUrl;
        this.httpClient = HttpClient.newBuilder()
                .connectTimeout(Duration.ofSeconds(10))
                .build();
        this.gson = createGson();
    }

    private Gson createGson() {
        return new GsonBuilder()
                .registerTypeAdapter(LocalDate.class, new LocalDateAdapter())
                .create();
    }

    // ==================== Owner API ====================

    public List<OwnerDto> getAllOwners() throws ApiException {
        return getList("/owners", new TypeToken<List<OwnerDto>>(){}.getType());
    }

    public OwnerDto getOwner(Long id) throws ApiException {
        return get("/owners/" + id, OwnerDto.class);
    }

    public OwnerDto createOwner(OwnerDto owner) throws ApiException {
        return post("/owners", owner, OwnerDto.class);
    }

    public OwnerDto updateOwner(Long id, OwnerDto owner) throws ApiException {
        return put("/owners/" + id, owner, OwnerDto.class);
    }

    public void deleteOwner(Long id) throws ApiException {
        delete("/owners/" + id);
    }

    public List<OwnerDto> searchOwners(String lastName) throws ApiException {
        String path = lastName != null && !lastName.isEmpty()
            ? "/owners?lastName=" + lastName
            : "/owners";
        return getList(path, new TypeToken<List<OwnerDto>>(){}.getType());
    }

    // ==================== Pet API ====================

    public List<PetDto> getPetsByOwner(Long ownerId) throws ApiException {
        return getList("/owners/" + ownerId + "/pets", new TypeToken<List<PetDto>>(){}.getType());
    }

    public PetDto getPet(Long ownerId, Long petId) throws ApiException {
        return get("/owners/" + ownerId + "/pets/" + petId, PetDto.class);
    }

    public PetDto createPet(Long ownerId, PetDto pet) throws ApiException {
        return post("/owners/" + ownerId + "/pets", pet, PetDto.class);
    }

    public PetDto updatePet(Long ownerId, Long petId, PetDto pet) throws ApiException {
        return put("/owners/" + ownerId + "/pets/" + petId, pet, PetDto.class);
    }

    public void deletePet(Long ownerId, Long petId) throws ApiException {
        delete("/owners/" + ownerId + "/pets/" + petId);
    }

    public List<PetTypeDto> getPetTypes() throws ApiException {
        return getList("/pettypes", new TypeToken<List<PetTypeDto>>(){}.getType());
    }

    // ==================== Visit API ====================

    public List<VisitDto> getVisitsByPet(Long ownerId, Long petId) throws ApiException {
        return getList("/owners/" + ownerId + "/pets/" + petId + "/visits",
                new TypeToken<List<VisitDto>>(){}.getType());
    }

    public VisitDto createVisit(Long ownerId, Long petId, VisitDto visit) throws ApiException {
        return post("/owners/" + ownerId + "/pets/" + petId + "/visits", visit, VisitDto.class);
    }

    // ==================== Vet API ====================

    public List<VetDto> getAllVets() throws ApiException {
        return getList("/vets", new TypeToken<List<VetDto>>(){}.getType());
    }

    public VetDto getVet(Long id) throws ApiException {
        return get("/vets/" + id, VetDto.class);
    }

    // ==================== HTTP Methods ====================

    private <T> T get(String path, Class<T> responseType) throws ApiException {
        try {
            HttpRequest request = HttpRequest.newBuilder()
                    .uri(URI.create(baseUrl + path))
                    .header("Accept", "application/json")
                    .GET()
                    .build();

            HttpResponse<String> response = httpClient.send(request, HttpResponse.BodyHandlers.ofString());
            handleErrorResponse(response);
            return gson.fromJson(response.body(), responseType);
        } catch (IOException | InterruptedException e) {
            throw new ApiException("Failed to GET " + path, e);
        }
    }

    private <T> List<T> getList(String path, Type listType) throws ApiException {
        try {
            HttpRequest request = HttpRequest.newBuilder()
                    .uri(URI.create(baseUrl + path))
                    .header("Accept", "application/json")
                    .GET()
                    .build();

            HttpResponse<String> response = httpClient.send(request, HttpResponse.BodyHandlers.ofString());
            handleErrorResponse(response);
            return gson.fromJson(response.body(), listType);
        } catch (IOException | InterruptedException e) {
            throw new ApiException("Failed to GET list " + path, e);
        }
    }

    private <T> T post(String path, Object body, Class<T> responseType) throws ApiException {
        try {
            HttpRequest request = HttpRequest.newBuilder()
                    .uri(URI.create(baseUrl + path))
                    .header("Content-Type", "application/json")
                    .header("Accept", "application/json")
                    .POST(HttpRequest.BodyPublishers.ofString(gson.toJson(body)))
                    .build();

            HttpResponse<String> response = httpClient.send(request, HttpResponse.BodyHandlers.ofString());
            handleErrorResponse(response);
            if (response.body() != null && !response.body().isEmpty()) {
                return gson.fromJson(response.body(), responseType);
            }
            return null;
        } catch (IOException | InterruptedException e) {
            throw new ApiException("Failed to POST " + path, e);
        }
    }

    private <T> T put(String path, Object body, Class<T> responseType) throws ApiException {
        try {
            HttpRequest request = HttpRequest.newBuilder()
                    .uri(URI.create(baseUrl + path))
                    .header("Content-Type", "application/json")
                    .header("Accept", "application/json")
                    .PUT(HttpRequest.BodyPublishers.ofString(gson.toJson(body)))
                    .build();

            HttpResponse<String> response = httpClient.send(request, HttpResponse.BodyHandlers.ofString());
            handleErrorResponse(response);
            if (response.body() != null && !response.body().isEmpty()) {
                return gson.fromJson(response.body(), responseType);
            }
            return null;
        } catch (IOException | InterruptedException e) {
            throw new ApiException("Failed to PUT " + path, e);
        }
    }

    private void delete(String path) throws ApiException {
        try {
            HttpRequest request = HttpRequest.newBuilder()
                    .uri(URI.create(baseUrl + path))
                    .DELETE()
                    .build();

            HttpResponse<String> response = httpClient.send(request, HttpResponse.BodyHandlers.ofString());
            handleErrorResponse(response);
        } catch (IOException | InterruptedException e) {
            throw new ApiException("Failed to DELETE " + path, e);
        }
    }

    private void handleErrorResponse(HttpResponse<String> response) throws ApiException {
        if (response.statusCode() >= 400) {
            throw new ApiException("API error: " + response.statusCode() + " - " + response.body());
        }
    }

    // ==================== LocalDate Adapter ====================

    private static class LocalDateAdapter implements JsonSerializer<LocalDate>, JsonDeserializer<LocalDate> {
        private static final DateTimeFormatter formatter = DateTimeFormatter.ISO_LOCAL_DATE;

        @Override
        public JsonElement serialize(LocalDate date, Type typeOfSrc, JsonSerializationContext context) {
            return new JsonPrimitive(date.format(formatter));
        }

        @Override
        public LocalDate deserialize(JsonElement json, Type typeOfT, JsonDeserializationContext context)
                throws JsonParseException {
            return LocalDate.parse(json.getAsString(), formatter);
        }
    }
}
